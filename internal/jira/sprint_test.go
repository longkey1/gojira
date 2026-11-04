package jira

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/longkey1/gojira/internal/config"
)

func TestListSprints(t *testing.T) {
	t.Run("follows pagination until isLast", func(t *testing.T) {
		var startAts []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %q, want GET", r.Method)
			}
			if r.URL.Path != "/rest/agile/1.0/board/7417/sprint" {
				t.Errorf("path = %q, want /rest/agile/1.0/board/7417/sprint", r.URL.Path)
			}
			if got := r.URL.Query().Get("state"); got != "active,closed" {
				t.Errorf("state = %q, want active,closed", got)
			}
			startAt := r.URL.Query().Get("startAt")
			startAts = append(startAts, startAt)

			switch startAt {
			case "0":
				_, _ = fmt.Fprint(w, `{"maxResults":2,"startAt":0,"isLast":false,"values":[
					{"id":1,"state":"closed","name":"S1","startDate":"2026-09-01T00:00:00.000Z","endDate":"2026-09-14T00:00:00.000Z","completeDate":"2026-09-14T01:00:00.000Z","originBoardId":7417},
					{"id":2,"state":"closed","name":"S2","originBoardId":7417}]}`)
			case "2":
				_, _ = fmt.Fprint(w, `{"maxResults":2,"startAt":2,"isLast":true,"values":[
					{"id":3,"state":"active","name":"S3","startDate":"2026-10-28T00:00:00.000Z","originBoardId":7417}]}`)
			default:
				t.Errorf("unexpected startAt %q", startAt)
				w.WriteHeader(http.StatusBadRequest)
			}
		}))
		t.Cleanup(srv.Close)

		client := NewClient(&config.Config{BaseURL: srv.URL})
		sprints, err := client.ListSprints(context.Background(), 7417, []string{"active", "closed"})
		if err != nil {
			t.Fatalf("ListSprints() error = %v", err)
		}

		if strings.Join(startAts, ",") != "0,2" {
			t.Errorf("startAt sequence = %v, want [0 2]", startAts)
		}
		if len(sprints) != 3 {
			t.Fatalf("got %d sprints, want 3", len(sprints))
		}
		if sprints[0].CompleteDate != "2026-09-14T01:00:00.000Z" || sprints[0].OriginBoardID != 7417 {
			t.Errorf("sprints[0] = %+v, want completeDate and originBoardId decoded", sprints[0])
		}
		if sprints[2].Name != "S3" || sprints[2].State != "active" || sprints[2].StartDate != "2026-10-28T00:00:00.000Z" {
			t.Errorf("sprints[2] = %+v, want active S3", sprints[2])
		}
	})

	t.Run("omits state when no states given", func(t *testing.T) {
		client, captured := newTestServer(t, http.StatusOK, `{"isLast":true,"values":[]}`)

		sprints, err := client.ListSprints(context.Background(), 1, nil)
		if err != nil {
			t.Fatalf("ListSprints() error = %v", err)
		}
		if captured.Query.Has("state") {
			t.Errorf("state query = %q, want absent", captured.Query.Get("state"))
		}
		if sprints == nil || len(sprints) != 0 {
			t.Errorf("sprints = %v, want empty non-nil slice", sprints)
		}
	})

	t.Run("stops on empty page without isLast", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			_, _ = fmt.Fprint(w, `{"isLast":false,"values":[]}`)
		}))
		t.Cleanup(srv.Close)

		client := NewClient(&config.Config{BaseURL: srv.URL})
		if _, err := client.ListSprints(context.Background(), 1, nil); err != nil {
			t.Fatalf("ListSprints() error = %v", err)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})

	t.Run("returns error on non-200", func(t *testing.T) {
		client, _ := newTestServer(t, http.StatusNotFound, `{"errorMessages":["Board does not exist"]}`)

		_, err := client.ListSprints(context.Background(), 1, nil)
		if err == nil || !strings.Contains(err.Error(), strconv.Itoa(http.StatusNotFound)) {
			t.Errorf("ListSprints() error = %v, want status 404 error", err)
		}
	})
}
