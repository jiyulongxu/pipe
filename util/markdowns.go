// Pipe - A small and beautiful blogging platform written in golang.
// Copyright (C) 2017, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package util

import (
	"crypto/md5"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/bluele/gcache"
	"github.com/hackebrot/turtle"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/vinta/pangu"
)

var markdownCache = gcache.New(1024).LRU().Build()

type MarkdownResult struct {
	ContentHTML  string
	AbstractText string
	ThumbURL     string
}

func Markdown(mdText string) *MarkdownResult {
	digest := md5.New()
	digest.Write([]byte(mdText))
	key := string(digest.Sum(nil))

	cached, err := markdownCache.Get(key)
	if nil == err {
		return cached.(*MarkdownResult)
	}

	mdText = emojify(mdText)
	mdTextBytes := []byte(mdText)
	unsafe := blackfriday.MarkdownCommon(mdTextBytes)
	contentHTML := string(bluemonday.UGCPolicy().AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9]+$")).OnElements("code").SanitizeBytes(unsafe))

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(contentHTML))
	doc.Find("img").Each(func(i int, ele *goquery.Selection) {
		src, _ := ele.Attr("src")
		ele.SetAttr("data-src", src)
		ele.RemoveAttr("src")
	})

	doc.Find("*").Contents().FilterFunction(func(i int, ele *goquery.Selection) bool {
		return "#text" == goquery.NodeName(ele)
	}).Each(func(i int, ele *goquery.Selection) {
		text := ele.Text()
		text = pangu.SpacingText(text)
		ele.ReplaceWithHtml(text)
	})

	contentHTML, _ = doc.Find("body").Html()

	text := doc.Text()
	runes := []rune{}
	for i, w := 0, 0; i < len(text); i += w {
		runeValue, width := utf8.DecodeRuneInString(text[i:])
		w = width

		if unicode.IsSpace(runeValue) {
			continue
		}

		runes = append(runes, runeValue)
		if 200 < len(runes) {
			break
		}
	}

	selection := doc.Find("img").First()
	thumbnailURL, _ := selection.Attr("src")
	if "" == thumbnailURL {
		thumbnailURL, _ = selection.Attr("data-src")
	}
	abstractText := strings.TrimSpace(runesToString(runes))
	abstractText = pangu.SpacingText(abstractText)

	ret := &MarkdownResult{
		ContentHTML:  contentHTML,
		AbstractText: abstractText,
		ThumbURL:     thumbnailURL,
	}
	markdownCache.Set(key, ret)

	return ret
}

func runesToString(runes []rune) (ret string) {
	for _, v := range runes {
		ret += string(v)
	}

	return
}

var emojiRegx = regexp.MustCompile(":[a-z_]+:")

func emojify(text string) string {
	return emojiRegx.ReplaceAllStringFunc(text, func(emojiASCII string) string {
		emojiASCII = strings.Replace(emojiASCII, ":", "", -1)
		emoji := turtle.Emojis[emojiASCII]
		if nil == emoji {
			logger.Warn("not found [" + emojiASCII + "]")

			return emojiASCII
		}

		return emoji.Char
	})
}
