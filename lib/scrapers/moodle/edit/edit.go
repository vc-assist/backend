package edit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"vcassist-backend/lib/scrapers/moodle/core"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("scrapers/moodle/edit")

type Course struct {
	Id   string
	Core *core.Client

	href      *url.URL
	isEditing bool
}

func NewCourse(ctx context.Context, id string, core *core.Client) (Course, error) {
	query := url.Values{}
	query.Add("id", id)
	href := &url.URL{
		Path:     "/course/view.php",
		RawQuery: query.Encode(),
	}
	absHref := core.BaseUrl.ResolveReference(href)

	if core.Sesskey == "" {
		return Course{}, fmt.Errorf("Your client must have a valid sesskey in order to edit course.")
	}
	return Course{Id: id, Core: core, href: absHref}, nil
}

func (c Course) ensureEditing(ctx context.Context) error {
	if c.isEditing {
		return nil
	}

	ctx, span := tracer.Start(ctx, "ensureEditing")
	defer span.End()

	redirects := 0
	c.Core.Http.SetRedirectPolicy(resty.RedirectPolicyFunc(
		func(req *http.Request, via []*http.Request) error {
			redirects++
			return nil
		},
	))
	defer c.Core.Http.SetRedirectPolicy(c.Core.DefaultRedirectPolicy())

	_, err := c.Core.Http.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"sesskey": c.Core.Sesskey,
			"id":      c.Id,
			"edit":    "on",
		}).
		Post("/course/view.php")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make request")
		return err
	}
	if redirects == 0 {
		span.SetStatus(codes.Error, "request didn't redirect")
		return fmt.Errorf("failed to enable editing, didn't redirect")
	}

	c.isEditing = true
	return nil
}

type Section struct {
	Name string
	Id   string
}

func (c Course) ListSections(ctx context.Context) ([]Section, error) {
	ctx, span := tracer.Start(ctx, "ListSections")
	defer span.End()

	err := c.ensureEditing(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to ensure editing mode is on")
		return nil, err
	}

	res, err := c.Core.Http.R().
		SetContext(ctx).
		Get(c.href.String())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make fetch request")
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(res.Body()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse html")
		return nil, err
	}

	var sections []Section
	doc.Find("li[data-sectionid]").Each(func(_ int, s *goquery.Selection) {
		id := s.AttrOr("data-id", "")
		if id == "" {
			return
		}
		nameAnchor := s.Find("h3 a[title]")
		name := strings.Trim(nameAnchor.Text(), " \n\t")
		sections = append(sections, Section{
			Id:   id,
			Name: name,
		})
	})

	return sections, nil
}

type action struct {
	Args       actionArgs `json:"args"`
	Index      int        `json:"index"`
	MethodName string     `json:"methodname"`
}

type actionArgs interface {
	actionArgs()
}

type actionList []action

func (a actionList) do(ctx context.Context, course Course) (*resty.Response, error) {
	ctx, span := tracer.Start(ctx, "action:do")
	defer span.End()

	if len(a) == 0 {
		err := fmt.Errorf("you must have at least one action to make a request")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	err := course.ensureEditing(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to ensure editing mode enabled")
		return nil, err
	}

	body, err := json.Marshal(a)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal body")
		return nil, err
	}

	res, err := course.Core.Http.R().
		SetContext(ctx).
		SetQueryParam("sesskey", course.Core.Sesskey).
		SetQueryParam("info", a[0].MethodName).
		SetBody(body).
		SetHeader("content-type", "application/json").
		Post("/lib/ajax/service.php")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make request")
		return nil, err
	}

	return res, nil
}

// "cd" stands for "create or delete"
type cdActionArgs struct {
	Action          string   `json:"action"`
	CourseId        string   `json:"courseid"`
	Ids             []string `json:"ids"`
	TargetSectionId string   `json:"targetsectionid,omitempty"`
}

func (cdActionArgs) actionArgs() {}

type sectionResponse []struct {
	Type   string `json:"name"`
	Fields struct {
		Id    string `json:"id"`
		Title string `json:"title"`
	} `json:"fields"`
}

// note: this does not return the new sections created, but all the sections after creating the new sections
func (c Course) CreateSections(ctx context.Context, lastSectionId string, count int) ([]Section, error) {
	ctx, span := tracer.Start(ctx, "CreateSection")
	defer span.End()

	if count <= 0 {
		err := fmt.Errorf("you must specify a count of at least 1 to add sections")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	act := action{
		Args: cdActionArgs{
			Action:          "section_add",
			Ids:             []string{},
			CourseId:        c.Id,
			TargetSectionId: lastSectionId,
		},
		Index:      0,
		MethodName: "core_courseformat_update_course",
	}
	actList := make(actionList, count)
	for i := 0; i < count; i++ {
		actList[i] = act
	}

	res, err := actList.do(ctx, c)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to do action")
		return nil, err
	}

	var responseJson []struct {
		Data string `json:"data"`
	}
	err = json.Unmarshal(res.Body(), &responseJson)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse response json")
		return nil, err
	}
	if len(responseJson) == 0 {
		err := fmt.Errorf("got empty response json")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	var sectionData sectionResponse
	err = json.Unmarshal([]byte(responseJson[0].Data), &sectionData)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse section data json")
		return nil, err
	}

	var sections []Section
	for _, entry := range sectionData {
		if entry.Type != "section" {
			continue
		}
		sections = append(sections, Section{
			Id:   entry.Fields.Id,
			Name: entry.Fields.Title,
		})
	}
	return sections, nil
}

// "r" stands for "rename"
type rActionArgs struct {
	Component string `json:"component"`
	Id        string `json:"itemid"`
	ItemType  string `json:"itemtype"`
	Value     string `json:"value"`
}

func (rActionArgs) actionArgs() {}

type RenameEntry struct {
	SectionId string
	NewName   string
}

func (c Course) RenameSections(ctx context.Context, entries []RenameEntry) error {
	ctx, span := tracer.Start(ctx, "RenameSection")
	defer span.End()

	err := c.ensureEditing(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to ensure editing mode enabled")
		return err
	}

	act := make(actionList, len(entries))
	for i, e := range entries {
		act[i] = action{
			Args: rActionArgs{
				Id:        e.SectionId,
				Component: "format_tiles",
				ItemType:  "sectionnamenl",
				Value:     e.NewName,
			},
			Index:      0,
			MethodName: "core_update_inplace_editable",
		}
	}

	_, err = act.do(ctx, c)
	return err
}

func (c Course) DeleteSections(ctx context.Context, sectionIds []string) error {
	ctx, span := tracer.Start(ctx, "RemoveSection")
	defer span.End()

	err := c.ensureEditing(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to ensure editing mode enabled")
		return err
	}

	actList := make(actionList, len(sectionIds))
	for i, id := range sectionIds {
		actList[i] = action{
			Args: cdActionArgs{
				Action:   "section_delete",
				CourseId: c.Id,
				Ids:      []string{id},
			},
			Index:      0,
			MethodName: "core_courseformat_update_course",
		}
	}
	_, err = actList.do(ctx, c)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to do action")
		return err
	}
	return nil
}
