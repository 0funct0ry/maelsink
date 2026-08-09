import { useNavigate } from 'react-router-dom'
import { Ghost } from 'lucide-react'
import Button from '../common/Button'

interface DeletedMessageStateProps {
  messageId?: string
}

export default function DeletedMessageState({ messageId }: DeletedMessageStateProps) {
  const navigate = useNavigate()

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 p-12 text-center">
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-surface-2">
        <Ghost className="h-8 w-8 text-text-tertiary" aria-hidden="true" />
      </div>
      <div>
        <h2 className="text-lg font-semibold text-text-primary">This message no longer exists</h2>
        <p className="mt-1 text-sm text-text-secondary">
          {messageId
            ? `The message "${messageId}" may have been deleted or cleared.`
            : 'It may have been deleted or cleared.'}
        </p>
      </div>
      <Button variant="secondary" onClick={() => navigate('/')}>
        Back to Inbox
      </Button>
    </div>
  )
}
