package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/imjasonmiller/godice"
	"golang.org/x/net/html"
)

type meeting struct {
	id       string
	members  []string
	date     string
	canceled bool
	location string
	entities []string
	subjects string
}

// Given a node, traverse all children, then apply the passed function to each.
func traverseNodes(node *html.Node, fn func(*html.Node)) {
	if node == nil {
		return
	}

	fn(node)

	for cur := node.FirstChild; cur != nil; cur = cur.NextSibling {
		traverseNodes(cur, fn)
	}
}

func meetingDate(sel *goquery.Selection) string {
	exp := regexp.MustCompile(`(?:\d{2}\/\d{2}\/\d{4})`)
	return exp.FindString(sel.Text())
}

func meetingCanceled(sel *goquery.Selection) bool {
	exp := regexp.MustCompile(`(?i:cancelled)`)
	return exp.MatchString(sel.Text())
}

func meetingLocation(sel *goquery.Selection) string {
	return strings.TrimSpace(sel.Text())
}

func meetingMembers(dep *[]member, sel *goquery.Selection) []string {
	result := []string{}

	memberNames := []string{}
	memberNameToID := map[string]string{}

	for _, member := range *dep {
		memberNames = append(memberNames, member.Name)
		memberNameToID[member.Name] = *member.ID
	}

	sel.Contents().EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		if sel.Is("br") {
			return false
		}

		name := strings.TrimSpace(sel.Text())

		if len(name) < 1 {
			return false
		}

		// Check if member exists.
		matches, err := godice.CompareStrings(name, memberNames)
		if err != nil {
			fmt.Println(name, err)
		}

		// fmt.Println("input", name, "best match:", matches.BestMatch.Text)

		if matches.BestMatch.Score < 0.75 {
			fmt.Printf("could not find \"%s\"; Best match was: \"%s\", with a score of: %.f\n", name, matches.BestMatch.Text, matches.BestMatch.Score)
			return false
		}

		if id, ok := memberNameToID[matches.BestMatch.Text]; ok {
			result = append(result, id)
		}

		return true
	})

	return result
}

func meetingEntities(sel *goquery.Selection) []string {
	entities := []string{}

	// Get entities from comment nodes in selection.
	traverseNodes(sel.Nodes[0], func(node *html.Node) {
		if node.Type == html.CommentNode {
			exp := regexp.MustCompile(`id=([0-9]+-[0-9]+)`)

			if id := exp.FindStringSubmatch(node.Data); id != nil {
				entities = append(entities, id[1])
			} else {
				entities = append(entities, "Unregistered")
			}
		}
	})

	return entities
}

func meetingSubjects(sel *goquery.Selection) string {
	return strings.TrimSpace(sel.Text())
}

// Functions byLeader and byMember reference "meetings" via a closure
func byLeader(meetings *[]meeting) func(int, *goquery.Selection) {
	return func(_ int, sel *goquery.Selection) {
		meeting := meeting{}

		sel.Find("td").Each(func(i int, sel *goquery.Selection) {
			switch i {
			case 0:
				meeting.date = meetingDate(sel)
				meeting.canceled = meetingCanceled(sel)
			case 1:
				meeting.location = meetingLocation(sel)
			case 2:
				meeting.entities = meetingEntities(sel)
			case 3:
				meeting.subjects = meetingSubjects(sel)
			}
		})

		*meetings = append(*meetings, meeting)
	}
}

func byMember(members *[]member, meetings *[]meeting) func(int, *goquery.Selection) {
	return func(_ int, sel *goquery.Selection) {
		meeting := meeting{}

		sel.Find("td").Each(func(i int, sel *goquery.Selection) {
			switch i {
			case 0:
				meeting.members = meetingMembers(members, sel)
			case 1:
				meeting.date = meetingDate(sel)
				meeting.canceled = meetingCanceled(sel)
			case 2:
				meeting.location = meetingLocation(sel)
			case 3:
				meeting.entities = meetingEntities(sel)
			case 4:
				meeting.subjects = meetingSubjects(sel)
			}
		})

		// fmt.Printf("%+v", meeting.members)

		*meetings = append(*meetings, meeting)
	}
}

func scrape(extract func(int, *goquery.Selection), host, path string) error {
	// Request document.
	res, err := http.Get(fmt.Sprint(host, path))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("requests are rate limited")
		}

		return fmt.Errorf("bad response from server: %s", res.Status)
	}

	defer res.Body.Close()

	// Parse response with goquery.
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %v", path, err)
	}

	// Iterate over table rows and extract data.
	doc.Find("#listMeetingsTable tbody tr").Each(extract)

	// Timeout to prevent rate limits.
	time.Sleep(250 * time.Millisecond)

	// Find next page href and recur.
	if next, ok := doc.Find(".pagelinks a img[alt='Next']").Parent().Attr("href"); ok {
		return scrape(extract, host, next)
	}

	return nil
}

func meetings(db *sql.DB) error {
	err := forEachDepartment(filepath.Join("database", "departments"), func(dep department) error {
		for _, l := range dep.Leaders {
			leaderMeetings := &[]meeting{}
			memberMeetings := &[]meeting{}

			host := "http://ec.europa.eu"

			if l.LeaderHostID != "" {
				path := fmt.Sprint("/transparencyinitiative/meetings/meeting.do?host=", l.LeaderHostID)
				if err := scrape(byLeader(leaderMeetings), host, path); err != nil {
					return err
				}
			}

			if l.MemberHostID != "" && len(dep.Members) > 0 {
				path := fmt.Sprint("/transparencyinitiative/meetings/meeting.do?host=", l.MemberHostID)
				if err := scrape(byMember(&dep.Members, memberMeetings), host, path); err != nil {
					return err
				}
			}

			err := bulkUpsertMeetings(leaderMeetings, memberMeetings)
			if err != nil {
				break
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func bulkUpsertMeetings(leaderMeetings, memberMeetings *[]meeting) error {
	fmt.Printf("leaderMeetings: %+v\nmemberMeetings: %+v", *leaderMeetings, *memberMeetings)
	return nil
}
