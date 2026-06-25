package forgejo

import (
	"net/http"
	"net/url"
	"strconv"
)

// Response wraps http.Response, adding pagination context.
type Response struct {
	*http.Response
	FirstPage int
	NextPage  int
	PrevPage  int
	LastPage  int
}

// ListOptions specifies pagination for list endpoints.
type ListOptions struct {
	Page     int
	PageSize int
}

func (l *ListOptions) applyToURL(u *url.URL) {
	q := u.Query()
	if l.Page > 0 {
		q.Set("page", strconv.Itoa(l.Page))
	}
	if l.PageSize > 0 {
		q.Set("limit", strconv.Itoa(l.PageSize))
	}
	u.RawQuery = q.Encode()
}
