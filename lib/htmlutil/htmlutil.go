package htmlutil

import (
	"bytes"
	"context"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	Href string
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

func GetAnchors(ctx context.Context, sel *goquery.Selection) []Anchor {
	ctx, span := tracer.Start(ctx, "GetAnchors")
	defer span.End()

	anchors := []Anchor{}
	for _, n := range sel.Nodes {
		href := ""
		for _, a := range n.Attr {
			if a.Key == "href" {
				href = a.Val
				break
			}
		}

		link, err := url.Parse(href)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "got error while parsing url")
			continue
		}

		name := GetText(n)
		name = removeNonPrintable(name)
		name = strings.Trim(name, " \t\n")
		name = innerWhitespace.ReplaceAllString(name, " ")

		linkStr := link.String()
		anchors = append(anchors, Anchor{
			Name: name,
			Href: linkStr,
		})
		span.AddEvent("anchor", trace.WithAttributes(
			attribute.String("name", name),
			attribute.String("url", linkStr),
		))
	}

	return anchors
}
