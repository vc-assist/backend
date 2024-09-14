package commands

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"unicode"
	devenv "vcassist-backend/dev/env"
	"vcassist-backend/lib/serviceutil"
	"vcassist-backend/services/vcmoodle/db"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"

	_ "modernc.org/sqlite"
)

var targetDb *string
var chapterTarget *int64

func init() {
	targetDb = testCmd.Flags().String("db", "<dev_state>/vcmoodle.db", "The database to write scrape results to.")
	chapterTarget = testCmd.Flags().Int64("chapter", -1, "The chapter to specifically test.")
	rootCmd.AddCommand(testCmd)
}

var testCmd = &cobra.Command{
	Use:   "test [--db <path/to/output.db>] [--chapter <chapter_id>]",
	Short: "Validates the result of a moodle scrape.",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := devenv.ResolvePath(*targetDb)
		if err != nil {
			serviceutil.Fatal("failed to resolve db path", err)
		}
		database, err := sql.Open("sqlite", path)
		if err != nil {
			serviceutil.Fatal("failed to open db", err)
		}
		defer database.Close()
		qry := db.New(database)

		chapters, err := GetAllChapters(cmd.Context(), qry)
		if err != nil {
			serviceutil.Fatal("get all chapters", err)
		}

		ValidateHeaders(chapters)
	},
}

type Chapter struct {
	ID         int64
	ResourceID int64
	Name       string
	Contents   *html.Node
}

func GetAllChapters(ctx context.Context, qry *db.Queries) ([]Chapter, error) {
	courses, err := qry.GetAllCourses(ctx)
	if err != nil {
		return nil, err
	}

	var result []Chapter

	for _, course := range courses {
		sections, err := qry.GetCourseSections(ctx, course.ID)
		if err != nil {
			return nil, err
		}
		if course.Name == "VC Assist" {
			continue
		}

		for _, section := range sections {
			resources, err := qry.GetSectionResources(ctx, db.GetSectionResourcesParams{
				CourseID:   course.ID,
				SectionIdx: section.Idx,
			})
			if err != nil {
				return nil, err
			}
			for _, resource := range resources {
				if resource.Type != int64(db.RESOURCE_BOOK) {
					continue
				}

				chapters, err := qry.GetResourceChapters(ctx, db.GetResourceChaptersParams{
					CourseID:    course.ID,
					SectionIdx:  section.Idx,
					ResourceIdx: resource.Idx,
				})
				if err != nil {
					return nil, err
				}

				for _, chapter := range chapters {
					if *chapterTarget > 0 && chapter.ID != *chapterTarget {
						continue
					}
					buff := bytes.NewBufferString(chapter.ContentHtml)
					doc, err := html.Parse(buff)
					if err != nil {
						return nil, err
					}
					result = append(result, Chapter{
						ID:         chapter.ID,
						ResourceID: resource.ID.Int64,
						Name:       chapter.Name,
						Contents:   doc,
					})
				}
			}
		}
	}

	return result, nil
}

var collapseWhitespaceRe = regexp.MustCompile(`\s\s+`)

func normalize(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, text)
	text = strings.Trim(text, " \t\n")
	text = collapseWhitespaceRe.ReplaceAllString(text, " ")
	text = strings.ToLower(text)
	return text
}

type matcher struct {
	re *regexp.Regexp
	// maximum amount of words in matched title, 0 means unspecified
	maxWordLength int
}

var sectionTitleMatchers = []matcher{
	{re: regexp.MustCompile(`(?: |^)hw:?`), maxWordLength: 8},
	{re: regexp.MustCompile(`homework`), maxWordLength: 8},
	{re: regexp.MustCompile(`assignments`), maxWordLength: 8},
	{re: regexp.MustCompile(`unit *#?[\da-z]?`)},
	{re: regexp.MustCompile(`day *#?[\da-z]?`)},
	{re: regexp.MustCompile(`objectives`), maxWordLength: 8},
	{re: regexp.MustCompile(`unit *standard`)},
	{re: regexp.MustCompile(`learning *outcome`)},
	{re: regexp.MustCompile(`biblical *integration`)},
	{re: regexp.MustCompile(`learning *outcome`)},
	{re: regexp.MustCompile(`classroom *activities`)},
}

func isSectionTitle(title string) bool {
	title = normalize(title)
	if len(title) >= 120 {
		return false
	}
	for _, match := range sectionTitleMatchers {
		if match.re.MatchString(title) && len(strings.Split(title, " ")) <= match.maxWordLength {
			return true
		}
	}
	return false
}

func findSectionTitles(root *html.Node, out *[]string) {
	if root.DataAtom == 0 && isSectionTitle(root.Data) {
		*out = append(*out, root.Data)
		return
	}

	child := root.FirstChild
	for child != nil {
		findSectionTitles(child, out)
		child = child.NextSibling
	}
}

var converter = md.NewConverter("", true, nil)

func ValidateHeaders(chapters []Chapter) {
	for _, chapter := range chapters {
		hasHomework := false
		chapterName := normalize(chapter.Name)

		var found []string
		findSectionTitles(chapter.Contents, &found)

		for _, text := range found {
			normalized := normalize(text)
			if chapterName == normalized || normalized == "" {
				continue
			}
			if strings.Contains(normalized, "homework") ||
				strings.Contains(normalized, "assignments") {
				hasHomework = true
			}
		}

		if !hasHomework {
			slog.Warn(
				"failed to find homework header",
				"resource", chapter.ResourceID,
				"chapter", chapter.ID,
			)
			doc := goquery.NewDocumentFromNode(chapter.Contents)
			rendered := converter.Convert(doc.Selection)
			fmt.Fprintln(os.Stderr, "\n------------------------------------\n")
			fmt.Fprintln(os.Stderr, rendered)
			fmt.Fprintln(os.Stderr, "\n------------------------------------\n")
		}
	}
}
