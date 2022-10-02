package emailrenderer

import (
	"bytes"
	"io"

	"golang.org/x/net/html"
)

type Renderer struct {
	doc *html.Node
}

// Constructor

func NewRenderer(reader io.Reader) (*Renderer, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, err
	}
	r := Renderer{
		doc: doc,
	}
	return &r, nil
}

// Rendering functions

func (r *Renderer) Render(writer io.Writer) error {
	return html.Render(writer, r.doc)
}

func (r *Renderer) String() (string, error) {
	var writer bytes.Buffer
	err := r.Render(&writer)
	if err != nil {
		return "", err
	}
	return writer.String(), nil
}

// Public mutations

func (r *Renderer) ReplaceTextByID(mappings map[string]string) {
	mutateNodes(r.doc, func(n *html.Node) { replaceTextByID(n, mappings) })
}

func (r *Renderer) ReplaceHrefByID(mappings map[string]string) {
	mutateNodes(r.doc, func(n *html.Node) { replaceHrefByID(n, mappings) })
}

// Private mutations

func replaceTextByID(node *html.Node, mappings map[string]string) {
	if node.Type == html.ElementNode {
		for id, text := range mappings {
			if hasID(node, id) {
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					node.RemoveChild(child)
				}
				content := html.Node{
					Type: html.TextNode,
					Data: text,
				}
				node.AppendChild(&content)
			}
		}
	}
}

func replaceHrefByID(node *html.Node, mappings map[string]string) {
	if isElement(node, "a") {
		for id, url := range mappings {
			if hasID(node, id) {
				href := findOrCreateAttribute(node, "href")
				href.Val = url
			}
		}
	}
}

// Helpers

func doAny[T any](s []T, f func(T) bool) bool {
	for _, e := range s {
		if f(e) {
			return true
		}
	}
	return false
}

func selectFirst[T any](sp *[]T, f func(T) bool) *T {
	s := *sp
	for i := 0; i < len(s); i++ {
		if f(s[i]) {
			return &s[i]
		}
	}
	return nil
}

func hasID(n *html.Node, id string) bool {
	return doAny(n.Attr, func(a html.Attribute) bool { return a.Key == "id" && a.Val == id })
}

func isElement(n *html.Node, elementName string) bool {
	return n.Type == html.ElementNode && n.Data == elementName
}

func findOrCreateAttribute(n *html.Node, key string) (result *html.Attribute) {
	result = selectFirst(&n.Attr, func(a html.Attribute) bool { return a.Key == key })
	if result == nil {
		n.Attr = append(n.Attr, html.Attribute{Key: key})
		result = &n.Attr[len(n.Attr)-1]
	}
	return
}

func mutateNodes(node *html.Node, f func(*html.Node)) {
	f(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		mutateNodes(child, f)
	}
}
