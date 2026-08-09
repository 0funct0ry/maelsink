import { useMessageStore } from '../../stores/useMessageStore'
import EmptyState from './EmptyState'
import MessageRow from './MessageRow'

interface MessageListProps {
  onOpenMessage: (id: string) => void
}

export default function MessageList({ onOpenMessage }: MessageListProps) {
  const messages = useMessageStore((state) => state.messages)
  const listStatus = useMessageStore((state) => state.listStatus)
  const listError = useMessageStore((state) => state.listError)

  if (listStatus === 'loading') {
    return (
      <div className="flex flex-col gap-2 p-4">
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="h-10 animate-pulse rounded-md bg-surface-2" />
        ))}
      </div>
    )
  }

  if (listStatus === 'error') {
    return (
      <div className="p-6 text-sm text-danger">{listError?.message ?? 'Failed to load messages.'}</div>
    )
  }

  if (messages.length === 0) {
    return <EmptyState />
  }

  return (
    <div>
      {messages.map((message) => (
        <MessageRow key={message.id} message={message} onOpen={() => onOpenMessage(message.id)} />
      ))}
    </div>
  )
}
