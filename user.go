package geddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// UserService handles communication with the user
// related methods of the Reddit API.
type UserService interface {
	Get(ctx context.Context, username string) (*User, *Response, error)
	GetMultipleByID(ctx context.Context, ids ...string) (map[string]*UserShort, *Response, error)
	UsernameAvailable(ctx context.Context, username string) (bool, *Response, error)

	Overview(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Comments, *Response, error)
	OverviewOf(ctx context.Context, username string, opts ...SearchOptionSetter) (*Posts, *Comments, *Response, error)

	Posts(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error)
	PostsOf(ctx context.Context, username string, opts ...SearchOptionSetter) (*Posts, *Response, error)

	Comments(ctx context.Context, opts ...SearchOptionSetter) (*Comments, *Response, error)
	CommentsOf(ctx context.Context, username string, opts ...SearchOptionSetter) (*Comments, *Response, error)

	Saved(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Comments, *Response, error)
	Upvoted(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error)
	Downvoted(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error)
	Hidden(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error)
	Gilded(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error)

	GetFriendship(ctx context.Context, username string) (*Friendship, *Response, error)
	Friend(ctx context.Context, username string) (*Friendship, *Response, error)
	Unfriend(ctx context.Context, username string) (*Response, error)

	Block(ctx context.Context, username string) (*Blocked, *Response, error)
	// BlockByID(ctx context.Context, id string) (*Blocked, *Response, error)
	Unblock(ctx context.Context, username string) (*Response, error)
	// UnblockByID(ctx context.Context, id string) (*Response, error)

	Trophies(ctx context.Context) (Trophies, *Response, error)
	TrophiesOf(ctx context.Context, username string) (Trophies, *Response, error)
}

// UserServiceOp implements the UserService interface.
type UserServiceOp struct {
	client *Client
}

var _ UserService = &UserServiceOp{}

// User represents a Reddit user.
type User struct {
	// this is not the full ID, watch out.
	ID      string     `json:"id,omitempty"`
	Name    string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	PostKarma    int `json:"link_karma"`
	CommentKarma int `json:"comment_karma"`

	IsFriend         bool `json:"is_friend"`
	IsEmployee       bool `json:"is_employee"`
	HasVerifiedEmail bool `json:"has_verified_email"`
	NSFW             bool `json:"over_18"`
	IsSuspended      bool `json:"is_suspended"`
}

// UserShort represents a Reddit user, but
// contains fewer pieces of information.
type UserShort struct {
	Name    string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	PostKarma    int `json:"link_karma"`
	CommentKarma int `json:"comment_karma"`

	NSFW bool `json:"profile_over_18"`
}

// Friendship represents a friend relationship.
type Friendship struct {
	ID       string     `json:"rel_id,omitempty"`
	Friend   string     `json:"name,omitempty"`
	FriendID string     `json:"id,omitempty"`
	Created  *Timestamp `json:"date,omitempty"`
}

// Blocked represents a blocked relationship.
type Blocked struct {
	Blocked   string     `json:"name,omitempty"`
	BlockedID string     `json:"id,omitempty"`
	Created   *Timestamp `json:"date,omitempty"`
}

type rootTrophyListing struct {
	Kind     string   `json:"kind,omitempty"`
	Trophies Trophies `json:"data"`
}

// Trophy is a Reddit award.
type Trophy struct {
	Name string `json:"name"`
}

// Trophies is a list of trophies.
type Trophies []Trophy

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *Trophies) UnmarshalJSON(b []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	trophies, ok := data["trophies"]
	if !ok {
		return errors.New("data does not contain trophies")
	}

	trophyList, ok := trophies.([]interface{})
	if !ok {
		return errors.New("unexpected type for list of trophies")
	}

	for _, trophyData := range trophyList {
		trophyInfo, ok := trophyData.(map[string]interface{})
		if !ok {
			continue
		}

		info, ok := trophyInfo["data"]
		if !ok {
			continue
		}

		infoBytes, err := json.Marshal(info)
		if err != nil {
			continue
		}

		var trophy Trophy
		err = json.Unmarshal(infoBytes, &trophy)
		if err != nil {
			continue
		}

		*t = append(*t, trophy)
	}

	return nil
}

// Get returns information about the user.
func (s *UserServiceOp) Get(ctx context.Context, username string) (*User, *Response, error) {
	path := fmt.Sprintf("user/%s/about", username)
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(userRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Data, resp, nil
}

// GetMultipleByID returns multiple users from their full IDs.
// The response body is a map where the keys are the IDs (if they exist), and the value is the user
func (s *UserServiceOp) GetMultipleByID(ctx context.Context, ids ...string) (map[string]*UserShort, *Response, error) {
	type query struct {
		IDs []string `url:"ids,omitempty,comma"`
	}

	path := "api/user_data_by_account_ids"
	path, err := addOptions(path, query{ids})
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(map[string]*UserShort)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return *root, resp, nil
}

// UsernameAvailable checks whether a username is available for registration.
func (s *UserServiceOp) UsernameAvailable(ctx context.Context, username string) (bool, *Response, error) {
	type query struct {
		User string `url:"user,omitempty"`
	}

	path := "api/username_available"
	path, err := addOptions(path, query{username})
	if err != nil {
		return false, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return false, nil, err
	}

	root := new(bool)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return false, resp, err
	}

	return *root, resp, nil
}

// Overview returns a list of the client's posts and comments.
func (s *UserServiceOp) Overview(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Comments, *Response, error) {
	return s.OverviewOf(ctx, s.client.Username, opts...)
}

// OverviewOf returns a list of the user's posts and comments.
func (s *UserServiceOp) OverviewOf(ctx context.Context, username string, opts ...SearchOptionSetter) (*Posts, *Comments, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/overview", username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, resp, err
	}

	return root.getPosts(), root.getComments(), resp, nil
}

// Posts returns a list of the client's posts.
func (s *UserServiceOp) Posts(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error) {
	return s.PostsOf(ctx, s.client.Username, opts...)
}

// PostsOf returns a list of the user's posts.
func (s *UserServiceOp) PostsOf(ctx context.Context, username string, opts ...SearchOptionSetter) (*Posts, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/submitted", username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.getPosts(), resp, nil
}

// Comments returns a list of the client's comments.
func (s *UserServiceOp) Comments(ctx context.Context, opts ...SearchOptionSetter) (*Comments, *Response, error) {
	return s.CommentsOf(ctx, s.client.Username, opts...)
}

// CommentsOf returns a list of the user's comments.
func (s *UserServiceOp) CommentsOf(ctx context.Context, username string, opts ...SearchOptionSetter) (*Comments, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/comments", username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.getComments(), resp, nil
}

// Saved returns a list of the user's saved posts and comments.
func (s *UserServiceOp) Saved(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Comments, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/saved", s.client.Username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, resp, err
	}

	return root.getPosts(), root.getComments(), resp, nil
}

// Upvoted returns a list of the user's upvoted posts.
func (s *UserServiceOp) Upvoted(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/upvoted", s.client.Username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.getPosts(), resp, nil
}

// Downvoted returns a list of the user's downvoted posts.
func (s *UserServiceOp) Downvoted(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/downvoted", s.client.Username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.getPosts(), resp, nil
}

// Hidden returns a list of the user's hidden posts.
func (s *UserServiceOp) Hidden(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/hidden", s.client.Username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.getPosts(), resp, nil
}

// Gilded returns a list of the user's gilded posts.
func (s *UserServiceOp) Gilded(ctx context.Context, opts ...SearchOptionSetter) (*Posts, *Response, error) {
	form := newSearchOptions(opts...)

	path := fmt.Sprintf("user/%s/gilded", s.client.Username)
	path = addQuery(path, form)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.getPosts(), resp, nil
}

// GetFriendship returns friendship details with the specified user.
// If the user is not your friend, it will return an error.
func (s *UserServiceOp) GetFriendship(ctx context.Context, username string) (*Friendship, *Response, error) {
	path := fmt.Sprintf("api/v1/me/friends/%s", username)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(Friendship)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Friend friends a user.
func (s *UserServiceOp) Friend(ctx context.Context, username string) (*Friendship, *Response, error) {
	type request struct {
		Username string `json:"name"`
	}

	path := fmt.Sprintf("api/v1/me/friends/%s", username)
	body := request{username}

	req, err := s.client.NewRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, nil, err
	}

	root := new(Friendship)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Unfriend unfriends a user.
func (s *UserServiceOp) Unfriend(ctx context.Context, username string) (*Response, error) {
	path := fmt.Sprintf("api/v1/me/friends/%s", username)
	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Block blocks a user.
func (s *UserServiceOp) Block(ctx context.Context, username string) (*Blocked, *Response, error) {
	path := "api/block_user"

	form := url.Values{}
	form.Set("name", username)

	req, err := s.client.NewPostForm(path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(Blocked)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// // BlockByID blocks a user via their full id.
// func (s *UserServiceOp) BlockByID(ctx context.Context, id string) (*Blocked, *Response, error) {
// 	path := "api/block_user"

// 	form := url.Values{}
// 	form.Set("account_id", id)

// 	req, err := s.client.NewPostForm(path, form)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	root := new(Blocked)
// 	resp, err := s.client.Do(ctx, req, root)
// 	if err != nil {
// 		return nil, resp, err
// 	}

// 	return root, resp, nil
// }

// Unblock unblocks a user.
func (s *UserServiceOp) Unblock(ctx context.Context, username string) (*Response, error) {
	selfID, err := s.client.GetRedditID(ctx)
	if err != nil {
		return nil, err
	}

	path := "api/unfriend"

	form := url.Values{}
	form.Set("name", username)
	form.Set("type", "enemy")
	form.Set("container", selfID)

	req, err := s.client.NewPostForm(path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// // UnblockByID unblocks a user via their full id.
// func (s *UserServiceOp) UnblockByID(ctx context.Context, id string) (*Response, error) {
// 	selfID, err := s.client.GetRedditID(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	path := "api/unfriend"

// 	form := url.Values{}
// 	form.Set("id", id)
// 	form.Set("type", "enemy")
// 	form.Set("container", selfID)

// 	req, err := s.client.NewPostForm(path, form)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return s.client.Do(ctx, req, nil)
// }

// Trophies returns a list of your trophies.
func (s *UserServiceOp) Trophies(ctx context.Context) (Trophies, *Response, error) {
	return s.TrophiesOf(ctx, s.client.Username)
}

// TrophiesOf returns a list of the specified user's trophies.
func (s *UserServiceOp) TrophiesOf(ctx context.Context, username string) (Trophies, *Response, error) {
	path := fmt.Sprintf("api/v1/user/%s/trophies", username)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootTrophyListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Trophies, resp, nil
}
