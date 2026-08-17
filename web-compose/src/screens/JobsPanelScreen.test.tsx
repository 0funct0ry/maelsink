import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import JobsPanelScreen from './JobsPanelScreen'
import * as jobsApi from '../lib/composeJobsApi'
import { useConnectionStore } from '../stores/useConnectionStore'
import { useJobsStore } from '../stores/useJobsStore'

vi.mock('../lib/composeJobsApi', async () => {
  const actual = await vi.importActual<typeof jobsApi>('../lib/composeJobsApi')
  return { ...actual, startJob: vi.fn(), cancelJob: vi.fn(), listJobs: vi.fn(), openJobStream: vi.fn() }
})

beforeEach(() => {
  useConnectionStore.setState({ status: 'green', lastChecked: null, lastError: null })
  useJobsStore.setState({ jobs: [] })
  vi.mocked(jobsApi.listJobs).mockResolvedValue([])
  vi.mocked(jobsApi.openJobStream).mockReturnValue(() => {})
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('JobsPanelScreen', () => {
  it('renders a button for each job kind', () => {
    render(<JobsPanelScreen />)
    for (const kind of ['randmsg', 'intmsg', 'weirdmsg', 'blast', 'deluge']) {
      expect(screen.getByText(kind)).toBeInTheDocument()
    }
  })

  it('opens a modal for a kind and starts a job, wiring the WS stream', async () => {
    vi.mocked(jobsApi.startJob).mockResolvedValue({ jobId: 'job-1' })
    let tickHandler: ((tick: jobsApi.JobSnapshot) => void) | undefined
    vi.mocked(jobsApi.openJobStream).mockImplementation((_id, onTick) => {
      tickHandler = onTick
      return () => {}
    })

    render(<JobsPanelScreen />)

    fireEvent.click(screen.getByText('randmsg'))
    const startButtons = await screen.findAllByText('Start')
    fireEvent.click(startButtons[0])

    await waitFor(() => expect(jobsApi.startJob).toHaveBeenCalledWith('randmsg', expect.anything()))
    await waitFor(() => expect(jobsApi.openJobStream).toHaveBeenCalled())

    // The modal auto-closes right after Start, before the job finishes —
    // its Cancel/Start footer buttons should no longer be present.
    await waitFor(() => expect(screen.queryByText('Start')).not.toBeInTheDocument())

    tickHandler?.({
      jobId: 'job-1',
      kind: 'randmsg',
      status: 'completed',
      sent: 3,
      failed: 0,
      startedAt: new Date().toISOString(),
      elapsedSeconds: 1.2,
    })

    await waitFor(() => expect(screen.getAllByText('completed').length).toBeGreaterThan(0))
  })

  it('cancels a running job from the recent-jobs list', async () => {
    useJobsStore.setState({
      jobs: [
        {
          jobId: 'job-2',
          kind: 'intmsg',
          status: 'running',
          sent: 5,
          failed: 0,
          startedAt: new Date().toISOString(),
          elapsedSeconds: 3,
        },
      ],
    })
    vi.mocked(jobsApi.cancelJob).mockResolvedValue({
      jobId: 'job-2',
      kind: 'intmsg',
      status: 'cancelled',
      sent: 5,
      failed: 0,
      startedAt: new Date().toISOString(),
      elapsedSeconds: 3,
    })

    render(<JobsPanelScreen />)

    fireEvent.click(screen.getByText('Cancel'))

    await waitFor(() => expect(jobsApi.cancelJob).toHaveBeenCalledWith('job-2'))
  })

  it('clears recent jobs after confirming', async () => {
    useJobsStore.setState({
      jobs: [
        {
          jobId: 'job-3',
          kind: 'randmsg',
          status: 'completed',
          sent: 1,
          failed: 0,
          startedAt: new Date().toISOString(),
          elapsedSeconds: 0.1,
        },
      ],
    })

    render(<JobsPanelScreen />)

    fireEvent.click(screen.getByLabelText('Clear all recent jobs'))
    fireEvent.click(screen.getByText('Clear all'))

    await waitFor(() => expect(screen.getByText('No jobs started yet this session.')).toBeInTheDocument())
  })
})
