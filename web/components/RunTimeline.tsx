'use client';

import { useEffect, useRef, useState } from 'react';

type TimelineEvent = {
  type: 'status' | 'output' | 'complete' | 'failed';
  step: string;
  message: string;
  stream?: 'stdout' | 'stderr';
};

/**
 * RunTimeline subscribes to server-sent events and presents an append-only
 * activity feed so users can see each major step in run progression.
 */
export default function RunTimeline({
  runId,
  initialOutput = [],
  onComplete
}: {
  runId: string;
  initialOutput?: string[];
  onComplete?: () => void;
}) {
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [output, setOutput] = useState<string[]>(initialOutput);
  const onCompleteRef = useRef(onComplete);

  // Keep the callback reference fresh without forcing a new SSE subscription.
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  useEffect(() => {
    setOutput(initialOutput);
  }, [initialOutput]);

  useEffect(() => {
    let isClosed = false;
    setEvents([]);
    setOutput(initialOutput);

    const source = new EventSource(`/api/scrape-runs/${runId}/events`);

    source.onmessage = (event) => {
      if (isClosed) {
        return;
      }

      const payload = JSON.parse(event.data) as TimelineEvent;
      if (payload.type === 'output') {
        setOutput((current) => [...current, `[${payload.stream}] ${payload.message}`]);
      } else {
        setEvents((current) => [...current, payload]);
      }

      // When the run reaches review-ready status, notify parent and stop streaming.
      if (payload.step === 'needs_review') {
        onCompleteRef.current?.();
        isClosed = true;
        source.close();
      }
    };

    source.onerror = () => {
      isClosed = true;
      source.close();
    };

    return () => {
      isClosed = true;
      source.close();
      };
  }, [initialOutput, runId]);

  return (
    <section className="grid run-monitor">
      <div className="card">
        <h3>Live run timeline</h3>
        <ul>
          {events.map((event, index) => (
            <li key={`${event.step}-${index}`}>
              <strong>{event.step}</strong>: {event.message}
            </li>
          ))}
        </ul>
      </div>
      <div className="card">
        <h3>Agent console</h3>
        <pre className="agent-console">{output.length > 0 ? output.join('') : 'Waiting for agent output...'}</pre>
      </div>
    </section>
  );
}
