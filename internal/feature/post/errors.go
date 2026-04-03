package post

import (
	"net/http"

	"github.com/jarviisha/darkvoid/pkg/errors"
)

var (
	ErrPostNotFound      = errors.New("POST_NOT_FOUND", "post not found", http.StatusNotFound)
	ErrCommentNotFound   = errors.New("COMMENT_NOT_FOUND", "comment not found", http.StatusNotFound)
	ErrForbidden         = errors.New("FORBIDDEN", "you do not have permission to perform this action", http.StatusForbidden)
	ErrSelfLike          = errors.New("SELF_LIKE", "you cannot like your own post", http.StatusBadRequest)
	ErrSelfCommentLike   = errors.New("SELF_COMMENT_LIKE", "you cannot like your own comment", http.StatusBadRequest)
	ErrEmptyContent      = errors.New("EMPTY_CONTENT", "post content and media cannot both be empty", http.StatusBadRequest)
	ErrInvalidVisibility = errors.New("INVALID_VISIBILITY", "visibility must be one of: public, followers, private", http.StatusBadRequest)
)
