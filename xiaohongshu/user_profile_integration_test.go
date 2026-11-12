//go:build integration

package xiaohongshu

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// /pending 永不返回，模拟一直挂起的埋点请求。
func TestUserProfileIgnoresPendingRequests(t *testing.T) {
	page := newLoginTestPage(t, `<script>
        fetch('/pending');
        window.__INITIAL_STATE__ = {user: {
            userPageData: {value: {basicInfo: {nickname: 'tester'}, interactions: []}},
            activeTab: {value: {index: 0, query: 'note'}},
            notes: {value: [[{id: 'n1'}], [], []]},
            noteQueries: {value: [{hasMore: true}, {hasMore: true}, {hasMore: true}]}
        }};
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := NewUserProfileAction(page).UserProfile(ctx, "uid", "token", TabNotes)
	require.NoError(t, err)
	require.Equal(t, "tester", res.UserBasicInfo.Nickname)
	require.Len(t, res.Feeds, 1)
	require.Equal(t, "n1", res.Feeds[0].ID)
}

func TestUserProfileWaitsForActiveTabNotes(t *testing.T) {
	page := newLoginTestPage(t, `<script>
        window.__INITIAL_STATE__ = {user: {
            userPageData: {value: {basicInfo: {nickname: 'tester'}, interactions: []}},
            activeTab: {value: {index: 0, query: 'note'}},
            notes: {value: [[], [], []]},
            noteQueries: {value: [{hasMore: true}, {hasMore: true}, {hasMore: true}]}
        }};
        setTimeout(() => { window.__INITIAL_STATE__.user.notes.value[0] = [{id: 'late'}]; }, 500);
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := NewUserProfileAction(page).UserProfile(ctx, "uid", "token", TabNotes)
	require.NoError(t, err)
	require.Len(t, res.Feeds, 1)
	require.Equal(t, "late", res.Feeds[0].ID)
}

func TestUserProfileReadsPlainUserPageData(t *testing.T) {
	page := newLoginTestPage(t, `<script>
        window.__INITIAL_STATE__ = {user: {
            userPageData: {basicInfo: {nickname: 'plain'}, interactions: []},
            activeTab: {value: {index: 0, query: 'note'}},
            notes: {value: [[{id: 'n1'}], [], []]},
            noteQueries: {value: [{hasMore: true}, {hasMore: true}, {hasMore: true}]}
        }};
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := NewUserProfileAction(page).UserProfile(ctx, "uid", "token", TabNotes)
	require.NoError(t, err)
	require.Equal(t, "plain", res.UserBasicInfo.Nickname)
	require.Len(t, res.Feeds, 1)
}

func TestUserProfileAcceptsEmptyFinishedTab(t *testing.T) {
	page := newLoginTestPage(t, `<script>
        window.__INITIAL_STATE__ = {user: {
            userPageData: {value: {basicInfo: {nickname: 'tester'}, interactions: []}},
            activeTab: {value: {index: 1, query: 'fav'}},
            notes: {value: [[], [], []]},
            noteQueries: {value: [{hasMore: true}, {hasMore: false}, {hasMore: true}]}
        }};
    </script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := NewUserProfileAction(page).UserProfile(ctx, "uid", "token", TabFavorites)
	require.NoError(t, err)
	require.Empty(t, res.Feeds)
}
