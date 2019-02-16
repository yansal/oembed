package oembed

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	urlpkg "net/url"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type Data struct {
	Type    string
	Version string
	URL     string
	HTML    string `xml:"html"`
}

func Get(ctx context.Context, url string) (Data, error) {
	oembedURL, err := find(ctx, url)
	if err != nil {
		return Data{}, err
	}
	if oembedURL == "" {
		return Data{}, errors.New("couldn't find oembed URL")
	}

	// TODO: log request
	req, err := http.NewRequest(http.MethodGet, oembedURL, nil)
	if err != nil {
		return Data{}, errors.WithStack(err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return Data{}, errors.WithStack(err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Data{}, errors.WithStack(err)
	}

	var data Data
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/xml") {
		err = xml.Unmarshal(b, &data)
	} else {
		err = json.Unmarshal(b, &data)
	}
	return data, errors.Wrapf(err, "couldn't decode body %s", b)
}

func find(ctx context.Context, url string) (string, error) {
	// first, lookup in known providers
	for re, endpoint := range providerEndpoints {
		if !re.MatchString(url) {
			continue
		}
		if endpoint.Discovery {
			break // fallback to discover
		}

		oembedURL := *endpoint.URL
		oembedURL.RawQuery = urlpkg.Values{"url": []string{url}}.Encode()
		return oembedURL.String(), nil
	}

	// second, discover
	return discover(ctx, url)
}

func discover(ctx context.Context, url string) (string, error) {
	// TODO: log request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	var f func(*html.Node) string
	f = func(n *html.Node) string {
		if href := findHref(n); href != "" {
			return href
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if href := f(c); href != "" {
				return href
			}
		}
		return ""
	}
	return f(doc), nil
}

func findHref(n *html.Node) string {
	if n.Type != html.ElementNode || n.Data != "link" {
		return ""
	}

	var ok bool
	for i := range n.Attr {
		if n.Attr[i].Key == "type" && strings.HasSuffix(n.Attr[i].Val, "+oembed") {
			ok = true
			break
		}
	}
	if !ok {
		return ""
	}
	for i := range n.Attr {
		if n.Attr[i].Key == "href" {
			return n.Attr[i].Val
		}
	}
	return ""
}
