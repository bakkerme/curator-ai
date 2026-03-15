'use client';

import { useEffect, useRef, useState } from 'react';

type TimelineEvent = {
  step: string;
  message: string;
};

/**
 * RunTimeline subscribes to server-sent events and presents an append-only
 * activity feed so users can see each major step in run progression.
 */
export default function RunTimeline({ runId, onComplete }: { runId: string; onComplete?: () => void }) {
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const onCompleteRef = useRef(onComplete);

  // Keep the callback reference fresh without forcing a new SSE subscription.
  useEffect(() => {
    onCompleteRef.current = onComplete;
  }, [onComplete]);

  useEffect(() => {
    let isClosed = false;
    setEvents([]);

    const source = new EventSource(`/api/scrape-runs/${runId}/events`);

    source.onmessage = (event) => {
      if (isClosed) {
        return;
      }

      const payload = JSON.parse(event.data) as TimelineEvent;
      setEvents((current) => [...current, payload]);

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
  }, [runId]);

  return (
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
  );
}
