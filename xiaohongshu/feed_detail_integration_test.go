//go:build integration

package xiaohongshu

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	xhserrors "github.com/xpzouying/xiaohongshu-mcp/errors"
	"github.com/xpzouying/xiaohongshu-mcp/humanize"
)

func TestFeedDetailWaitsForTargetNote(t *testing.T) {
	page := newLoginTestPage(t, `<div id="changing"></div><script>
        window.__INITIAL_STATE__ = {note: {noteDetailMap: {
            target: {note: {}, comments: {list: [], hasMore: true}}
        }}};
        setInterval(() => document.querySelector('#changing').textContent = Date.now(), 50);
        setTimeout(() => {
            window.__INITIAL_STATE__.note.noteDetailMap.target = {
                note: {noteId: 'target', title: 'Ready', desc: 'Complete note',
                    imageList: [{urlDefault: 'https://example.com/image.jpg'}]},
                comments: {list: [], hasMore: false}
            };
        }, 500);
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	detail, err := NewFeedDetailAction(page).GetFeedDetail(ctx, "target", "token", false, DefaultCommentLoadConfig())
	require.NoError(t, err)
	require.Equal(t, "target", detail.Note.NoteID)
	require.Equal(t, "Ready", detail.Note.Title)
	require.Len(t, detail.Note.ImageList, 1)
	require.NotEqual(t, "complete", page.MustEval(`() => document.readyState`).Str())
}

func TestFeedDetailRejectsPlaceholder(t *testing.T) {
	page := newLoginTestPage(t, `<script>
        window.__INITIAL_STATE__ = {note: {noteDetailMap: {
            target: {note: {}, comments: {list: [], hasMore: true}}
        }}};
    </script>`)
	page.MustNavigate(getBaseURL())
	page.MustWait(`() => window.__INITIAL_STATE__ !== undefined`)
	_, err := NewFeedDetailAction(page).extractFeedDetail(page, "target")
	require.ErrorContains(t, err, "data not ready")
}

func TestFeedDetailMissingTargetRespectsCancellation(t *testing.T) {
	page := newLoginTestPage(t, `<script>
        window.__INITIAL_STATE__ = {note: {noteDetailMap: {
            other: {note: {noteId: 'other'}}
        }}};
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := NewFeedDetailAction(page).GetFeedDetail(ctx, "target", "token", false, DefaultCommentLoadConfig())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestFeedDetailGuestReturnsLoginError(t *testing.T) {
	page := newLoginTestPage(t, `<div class="login-container"></div><script>
        window.__INITIAL_STATE__ = {user: {userInfo: {guest: true}}, note: {noteDetailMap: {}}};
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := NewFeedDetailAction(page).GetFeedDetail(ctx, "target", "token", false, DefaultCommentLoadConfig())
	require.ErrorIs(t, err, xhserrors.ErrNotLoggedIn)
}

type feedDetailTestTiming struct{}

func (feedDetailTestTiming) Timing() humanize.TimingProfile {
	return humanize.TimingProfile{humanize.AfterNavigate: {Max: time.Millisecond}}
}

func TestFeedDetailWaitsForInitialComments(t *testing.T) {
	humanize.SetProvider(feedDetailTestTiming{})
	t.Cleanup(func() { humanize.SetProvider(humanize.DefaultProvider{}) })
	page := newLoginTestPage(t, `<div class="error-wrapper"></div><script>
		window.__INITIAL_STATE__ = {note: {noteDetailMap: {
			target: {
				note: {noteId: 'target', title: 'Ready', interactInfo: {commentCount: '3'}},
				comments: {list: [], cursor: '', hasMore: true}
			}
		}}};
		setTimeout(() => {
			window.__INITIAL_STATE__.note.noteDetailMap.target.comments = {
				list: [{id: 'first-comment', noteId: 'target', content: 'Initial comment'}],
				cursor: 'first-page', hasMore: true
			};
		}, 4000);
	</script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	detail, err := NewFeedDetailAction(page).GetFeedDetail(ctx, "target", "token", false, DefaultCommentLoadConfig())
	require.NoError(t, err)
	require.Len(t, detail.Comments.List, 1)
	require.Equal(t, "Initial comment", detail.Comments.List[0].Content)
	require.Equal(t, "first-page", detail.Comments.Cursor)
	require.True(t, detail.Comments.HasMore)
	require.NotEqual(t, "complete", page.MustEval(`() => document.readyState`).Str())
}

func TestFeedDetailAcceptsExplicitZeroComments(t *testing.T) {
	humanize.SetProvider(feedDetailTestTiming{})
	t.Cleanup(func() { humanize.SetProvider(humanize.DefaultProvider{}) })
	page := newLoginTestPage(t, `<div class="error-wrapper"></div><script>
		window.__INITIAL_STATE__ = {note: {noteDetailMap: {
			target: {
				note: {noteId: 'target', interactInfo: {commentCount: '0'}},
				comments: {list: [], cursor: '', hasMore: true}
			}
		}}};
	</script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	detail, err := NewFeedDetailAction(page).GetFeedDetail(ctx, "target", "token", false, DefaultCommentLoadConfig())
	require.NoError(t, err)
	require.Empty(t, detail.Comments.List)
	require.Equal(t, "0", detail.Note.InteractInfo.CommentCount)
}

func TestFeedDetailUnknownCommentCountWaits(t *testing.T) {
	humanize.SetProvider(feedDetailTestTiming{})
	t.Cleanup(func() { humanize.SetProvider(humanize.DefaultProvider{}) })
	for _, count := range []string{"undefined", "null", "''", "'unknown'"} {
		t.Run(count, func(t *testing.T) {
			page := newLoginTestPage(t, `<div class="error-wrapper"></div><script>
				window.__INITIAL_STATE__ = {note: {noteDetailMap: {
					target: {
						note: {noteId: 'target', interactInfo: {commentCount: `+count+`}},
						comments: {list: [], hasMore: true}
					}
				}}};
			</script>`)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err := NewFeedDetailAction(page).GetFeedDetail(ctx, "target", "token", false, DefaultCommentLoadConfig())
			require.ErrorIs(t, err, context.DeadlineExceeded)
		})
	}
}
