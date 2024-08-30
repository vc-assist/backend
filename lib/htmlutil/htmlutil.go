package htmlutil

import (
	"bytes"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/html"
)

var tracer = otel.Tracer("vcassist.lib.htmlutil")

func GetText(node *html.Node) string {
	var buffer bytes.Buffer
	getTextRecursive(node, &buffer)
	return buffer.String()
}

func getTextRecursive(node *html.Node, buffer *bytes.Buffer) {
	if node == nil {
		return
	}
	if node.Type == html.TextNode {
		buffer.WriteString(node.Data)
		return
	}
	child := node.FirstChild
	for child != nil {
		getTextRecursive(child, buffer)
		child = child.NextSibling
	}
}

type Anchor struct {
	Name string
	Url  *url.URL
}

var innerWhitespace = regexp.MustCompile(`\s\s+`)

func removeNonPrintable(s string) string {
	newStr := strings.Builder{}
	for _, c := range s {
		if unicode.IsPrint(c) {
			newStr.WriteRune(c)
		}
	}
	return newStr.String()
}

func GetAnchors(baseUrl *url.URL, sel *goquery.Selection) []Anchor {
	anchors := []Anchor{}
	for _, n := range sel.Nodes {
		href := ""
		for _, a := range n.Attr {
			if a.Key == "href" {
				href = a.Val
				break
			}
		}

		link, err := baseUrl.Parse(href)
		if err != nil {
			slog.Warn(
				"failed to resolve anchor href",
				"href", href,
				"base_url", baseUrl,
				"err", err,
			)
			continue
		}

		name := GetText(n)
		name = removeNonPrintable(name)
		name = strings.Trim(name, " \t\n")
		name = innerWhitespace.ReplaceAllString(name, " ")

		anchors = append(anchors, Anchor{
			Name: name,
			Url:  link,
		})
	}

	return anchors
}
