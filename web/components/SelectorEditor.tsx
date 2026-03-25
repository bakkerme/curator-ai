'use client';

import { useEffect, useState } from 'react';

type SelectorValues = Record<string, string>;

/**
 * SelectorEditor presents field-level selector values and allows users to edit
 * them before requesting a preview refresh from the backend.
 */
export default function SelectorEditor({
  initialValues,
  onPreview
}: {
  initialValues: SelectorValues;
  onPreview: (nextValues: SelectorValues) => Promise<void>;
}) {
  const [values, setValues] = useState<SelectorValues>(initialValues);

  // Keep local editable state aligned with backend refreshes/new run payloads.
  useEffect(() => {
    setValues(initialValues);
  }, [initialValues]);

  return (
    <section className="card grid">
      <h3>Edit selectors</h3>
      {Object.entries(values).map(([key, value]) => (
        <label key={key}>
          {key}
          <input
            value={value}
            onChange={(event) =>
              setValues((current) => ({
                ...current,
                [key]: event.target.value
              }))
            }
          />
        </label>
      ))}
      <button type="button" onClick={() => onPreview(values)}>
        Re-run preview
      </button>
    </section>
  );
}
