// client.go contains all the logic for just scraping moodle itself, it does not contain
// any extra logic to account for how valley specifically uses moodle.

package moodle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
	"vcassist-backend/internal/components/assert"
	"vcassist-backend/internal/components/telemetry"
	"vcassist-backend/pkg/htmlutil"

	cloudflarebp "github.com/DaRealFreak/cloudflare-bp-go"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

const (
	report_client_get_sesskey             = "client.get-sesskey"
	report_client_login_username_password = "client.login-username-password"
	report_client_get_courses             = "client.get-courses"
	report_client_get_sections            = "client.get-sections"
	report_client_get_resources           = "client.get-resources"
	report_client_get_chapters            = "client.get-chapters"
	report_client_get_chapter_content     = "client.get-chapter-content"
)

type client struct {
	BaseUrl *url.URL
	Http    *resty.Client
	Sesskey string

	tel telemetry.API
}

func newClient(baseUrl string, tel telemetry.API) (*client, error) {
	assert.NotNil(tel)

	tel = telemetry.NewScopedAPI("moodle_scraper", tel)

	parsedBaseUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	httpClient := resty.New()
	httpClient.SetBaseURL(baseUrl)
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	httpClient.SetCookieJar(jar)
	httpClient.GetClient().Transport = cloudflarebp.AddCloudFlareByPass(httpClient.GetClient().Transport)

	httpClient.SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	httpClient.SetRedirectPolicy(resty.DomainCheckRedirectPolicy(parsedBaseUrl.Hostname()))
	httpClient.SetTimeout(time.Second * 30)

	// 2 requests max per second
	// max burst >= 2 just means that no requests will be dropped
	rateLimiter := rate.NewLimiter(2, 2)
	httpClient.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		err = rateLimiter.Wait(req.Context())
		if err != nil {
			return err
		}
		return nil
	})

	telemetry.InstrumentResty(httpClient, tel)

	c := &client{
		BaseUrl: parsedBaseUrl,
		Http:    httpClient,
		tel:     tel,
	}
	return c, nil
}

var moodleConfigRegex = regexp.MustCompile(`(?m)M\.cfg *= *(.+?);`)

func (c *client) getSesskey(doc *goquery.Document) string {
	for _, script := range doc.Find("script").Nodes {
		text := htmlutil.GetText(script)
		if !strings.HasPrefix(strings.Trim(text, " \t\n"), "//<![CDATA") {
			continue
		}
		groups := moodleConfigRegex.FindStringSubmatch(text)
		if len(groups) < 2 {
			continue
		}

		var cfg struct {
			Sesskey string `json:"sesskey"`
		}
		err := json.Unmarshal([]byte(groups[1]), &cfg)
		if err != nil {
			c.tel.ReportBroken(
				report_client_get_sesskey,
				fmt.Errorf("unmarshal moodle config: %w", err),
			)
			return ""
		}
		return cfg.Sesskey
	}

	return ""
}

func (c *client) LoginUsernamePassword(ctx context.Context, username, password string) error {
	loginError := func(err error) error {
		return fmt.Errorf("moodle scraper: login failed: %w", err)
	}

	res, err := c.Http.R().
		SetContext(ctx).
		Get("/login/index.php")
	if err != nil {
		c.tel.ReportBroken(
			report_client_login_username_password,
			fmt.Errorf("not-logged-in page request: %w", err),
		)
		return loginError(err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_login_username_password,
			fmt.Errorf("parse not-logged-in page: %w", err),
		)
		return loginError(err)
	}

	logintoken := doc.Find("input[name=logintoken]").AttrOr("value", "")
	if logintoken == "" {
		err := fmt.Errorf("could not find login token")
		c.tel.ReportBroken(
			report_client_login_username_password,
			err,
		)
		return loginError(err)
	}

	res, err = c.Http.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"logintoken": logintoken,
			"username":   username,
			"password":   password,
		}).
		Post("/login/index.php")
	if err != nil {
		c.tel.ReportBroken(
			report_client_login_username_password,
			fmt.Errorf("login request: %w", err),
		)
		return loginError(err)
	}

	res, err = c.Http.R().
		SetContext(ctx).
		Get("/")
	if err != nil {
		c.tel.ReportBroken(
			report_client_login_username_password,
			fmt.Errorf("request dashboard: %w", err),
		)
		return loginError(err)
	}
	doc, err = goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_login_username_password,
			fmt.Errorf("parse dashboard page: %w", err),
		)
		return loginError(err)
	}

	if len(doc.Find("span.avatar.current").Nodes) == 0 {
		c.tel.ReportWarning(
			report_client_login_username_password,
			fmt.Errorf("test login: could not find span.avatar.current"),
		)
		return fmt.Errorf("moodle scraper: login failed")
	}

	c.Sesskey = c.getSesskey(doc)
	return nil
}

func (c client) Courses(ctx context.Context) ([]Course, error) {
	c.tel.ReportDebug("get courses")

	res, err := c.Http.R().
		SetContext(ctx).
		Get("/index.php")
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_courses,
			fmt.Errorf("fetch: %w", err),
		)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_courses,
			fmt.Errorf("parse: %w", err),
		)
		return nil, err
	}

	anchors := htmlutil.GetAnchors(res.Request.RawRequest.URL, doc.Find("ul.unlist a"))

	return coursesFromAnchors(anchors), nil
}

func (c client) Sections(ctx context.Context, course Course) ([]Section, error) {
	endpoint := course.Url.String()
	c.tel.ReportDebug(report_client_get_sections, endpoint)

	res, err := c.Http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_sections,
			fmt.Errorf("fetch: %w", err),
			endpoint,
		)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_sections,
			fmt.Errorf("parse: %w", err),
			endpoint,
		)
		return nil, err
	}

	anchors := htmlutil.GetAnchors(course.Url, doc.Find(".course-content a.nav-link"))

	return sectionsFromAnchors(anchors), nil
}

func (c client) Resources(ctx context.Context, section Section) ([]Resource, error) {
	assert.NotNil(section.Url)

	endpoint := section.Url.String()

	c.tel.ReportDebug(report_client_get_resources, endpoint)

	res, err := c.Http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_resources,
			fmt.Errorf("fetch: %w", err),
			endpoint,
		)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_resources,
			fmt.Errorf("parse: %w", err),
			endpoint,
		)
		return nil, err
	}

	infoHtml, err := doc.Find("div[data-for=sectioninfo]").Html()
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_resources,
			fmt.Errorf("serialize section info: %w", err),
			endpoint,
		)
	}

	anchors := htmlutil.GetAnchors(section.Url, doc.Find("li.activity a"))
	resources := resourcesFromAnchors(anchors)
	if infoHtml != "" {
		resources = append([]Resource{{
			Type: RESOURCE_HTML_AREA,
			Name: infoHtml,
		}}, resources...)
	}

	return resources, nil
}

func (c client) Chapters(ctx context.Context, resource Resource) ([]Chapter, error) {
	endpoint := resource.Url.String()

	c.tel.ReportDebug(report_client_get_chapters, endpoint)

	res, err := c.Http.R().
		SetContext(ctx).
		Get(endpoint)
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_chapters,
			fmt.Errorf("fetch: %w", err),
			endpoint,
		)
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_chapters,
			fmt.Errorf("fetch: %w", err),
			endpoint,
		)
		return nil, err
	}

	tableOfContents := htmlutil.GetAnchors(resource.Url, doc.Find("div.columnleft li a"))

	currentChapter := doc.Find("div.columnleft li strong").Text()

	// the first chapter you click on doesn't give you its chapter id so you have to
	// rummage for it in this weird corner
	printUrl, exists := doc.Find("li[data-key=printchapter] a").First().Attr("href")
	if exists {
		parsed, err := url.Parse(printUrl)
		if err != nil {
			c.tel.ReportBroken(
				report_client_get_chapters,
				fmt.Errorf("parse chapter url: %w", err),
				endpoint,
			)
		} else {
			values := resource.Url.Query()
			values.Add("chapterid", parsed.Query().Get("chapterid"))
			resource.Url.RawQuery = values.Encode()
		}
	}

	anchors := append(tableOfContents, htmlutil.Anchor{
		Url:  resource.Url,
		Name: currentChapter,
	})

	return chaptersFromAnchors(anchors), nil
}

func (c client) ChapterContent(ctx context.Context, chapter Chapter) (string, error) {
	endpoint := chapter.Url

	c.tel.ReportDebug(report_client_get_chapter_content, endpoint.String())

	res, err := c.Http.R().
		SetContext(ctx).
		Get(endpoint.String())
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_chapter_content,
			fmt.Errorf("fetch: %w", err),
			endpoint.String(),
		)
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_chapter_content,
			fmt.Errorf("parse: %w", err),
			endpoint.String(),
		)
		return "", err
	}

	contents, err := doc.Find("div[role=main] div.box").Html()
	if err != nil {
		c.tel.ReportBroken(
			report_client_get_chapter_content,
			fmt.Errorf("serialize content: %w", err),
			endpoint.String(),
		)
		return "", err
	}

	return contents, nil
}
