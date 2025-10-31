import { useId, useState } from "react";
import type { Job } from "../types/job";

type Props = {
  initialType?: Job["type"];
  initialConfig?: Record<string, string>;
  onSubmit: (type: Job["type"], config: Record<string, string>) => void;
  submitLabel?: string;
};

export default function JobTypeForm({ initialType = "email", initialConfig = {}, onSubmit, submitLabel = "Save" }: Props) {
  const uid = useId();
  const [type, setType] = useState<Job["type"]>(initialType);
  const jobTypes = ["email", "sync", "backup", "custom"];
  const [config, setConfig] = useState<Record<string, string>>(() => ({ ...initialConfig }));
  const [errors, setErrors] = useState<Record<string, string>>({});

  const updateConfig = (key: string, value: string) => {
    setConfig((c) => ({ ...c, [key]: value }));
  };

  const validate = (): boolean => {
    const e: Record<string, string> = {};
    if (type === "email") {
      if (!config.to) e.to = "Recipient is required";
      if (!config.subject) e.subject = "Subject is required";
    }
    if (type === "sync") {
      if (!config.source) e.source = "Source is required";
      if (!config.destination) e.destination = "Destination is required";
    }
    if (type === "backup") {
      if (!config.path) e.path = "Path is required";
      if (!config.destination) e.destination = "Destination is required";
    }
    if (type === "custom") {
      if (!config.command) e.command = "Command is required";
    }
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (!validate()) return;
    onSubmit(type, config);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
  <label htmlFor={`${uid}-job-type`} className="block text-sm font-medium text-gray-700 mb-1">Job Type</label>
        <select id={`${uid}-job-type`} value={type} onChange={(e) => setType(e.target.value as Job["type"]) } className="w-full px-3 py-2 border border-gray-300 rounded-lg">
          {jobTypes.includes(type) ? null : <option value={type}>{type}</option>}
          {jobTypes.map((t) => (
            <option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>
          ))}
        </select>
      </div>

      <div>
        {type === "email" && (
          <div className="space-y-2">
            <div>
              <label htmlFor={`${uid}-to`} className="block text-xs text-gray-600">To</label>
              <input id={`${uid}-to`} className="w-full px-2 py-1 border rounded" value={config.to || ""} onChange={(e) => updateConfig("to", e.target.value)} />
              {errors.to && <div className="text-xs text-red-500">{errors.to}</div>}
            </div>
            <div>
              <label htmlFor={`${uid}-subject`} className="block text-xs text-gray-600">Subject</label>
              <input id={`${uid}-subject`} className="w-full px-2 py-1 border rounded" value={config.subject || ""} onChange={(e) => updateConfig("subject", e.target.value)} />
              {errors.subject && <div className="text-xs text-red-500">{errors.subject}</div>}
            </div>
            <div>
              <label htmlFor={`${uid}-body`} className="block text-xs text-gray-600">Body</label>
              <textarea id={`${uid}-body`} className="w-full px-2 py-1 border rounded" rows={3} value={config.body || ""} onChange={(e) => updateConfig("body", e.target.value)} />
            </div>
          </div>
        )}

        {type === "sync" && (
          <div className="space-y-2">
            <div>
              <label htmlFor={`${uid}-source`} className="block text-xs text-gray-600">Source</label>
              <input id={`${uid}-source`} className="w-full px-2 py-1 border rounded" value={config.source || ""} onChange={(e) => updateConfig("source", e.target.value)} />
              {errors.source && <div className="text-xs text-red-500">{errors.source}</div>}
            </div>
            <div>
              <label htmlFor={`${uid}-destination`} className="block text-xs text-gray-600">Destination</label>
              <input id={`${uid}-destination`} className="w-full px-2 py-1 border rounded" value={config.destination || ""} onChange={(e) => updateConfig("destination", e.target.value)} />
              {errors.destination && <div className="text-xs text-red-500">{errors.destination}</div>}
            </div>
          </div>
        )}

        {type === "backup" && (
          <div className="space-y-2">
            <div>
              <label htmlFor={`${uid}-path`} className="block text-xs text-gray-600">Path</label>
              <input id={`${uid}-path`} className="w-full px-2 py-1 border rounded" value={config.path || ""} onChange={(e) => updateConfig("path", e.target.value)} />
              {errors.path && <div className="text-xs text-red-500">{errors.path}</div>}
            </div>
            <div>
              <label htmlFor={`${uid}-destination-2`} className="block text-xs text-gray-600">Destination</label>
              <input id={`${uid}-destination-2`} className="w-full px-2 py-1 border rounded" value={config.destination || ""} onChange={(e) => updateConfig("destination", e.target.value)} />
              {errors.destination && <div className="text-xs text-red-500">{errors.destination}</div>}
            </div>
          </div>
        )}

        {type === "custom" && (
          <div>
            <label htmlFor={`${uid}-command`} className="block text-xs text-gray-600">Command</label>
            <input id={`${uid}-command`} className="w-full px-2 py-1 border rounded" value={config.command || ""} onChange={(e) => updateConfig("command", e.target.value)} />
            {errors.command && <div className="text-xs text-red-500">{errors.command}</div>}
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <button type="button" onClick={handleSubmit} className="bg-blue-600 text-white py-2 px-4 rounded">{submitLabel}</button>
        <button type="button" onClick={() => { setConfig({}); setErrors({}); }} className="bg-gray-100 px-4 rounded">Reset</button>
      </div>
    </form>
  );
}
