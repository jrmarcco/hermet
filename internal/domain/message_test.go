package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationIDRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("single chat id is order independent", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, SingleConversationID(1, 2), SingleConversationID(2, 1))
		require.Equal(t, "s:1:2", SingleConversationID(2, 1))

		// 单聊会话不应被解析为群聊会话。
		_, ok := ParseGroupConversationID(SingleConversationID(1, 2))
		require.False(t, ok)
	})

	t.Run("group chat id round trip", func(t *testing.T) {
		t.Parallel()

		const groupID = uint64(42)

		conversationID := GroupConversationID(groupID)
		require.Equal(t, "g:42", conversationID)

		parsed, ok := ParseGroupConversationID(conversationID)
		require.True(t, ok)
		require.Equal(t, groupID, parsed)
	})

	t.Run("invalid conversation id", func(t *testing.T) {
		t.Parallel()

		for _, id := range []string{"", "g:", "g:abc", "g:42junk", "s:1:2", "x:1"} {
			_, ok := ParseGroupConversationID(id)
			require.False(t, ok, "conversation id [ %s ] should not parse", id)
		}
	})
}
