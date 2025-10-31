import { useMutation, useQueryClient } from "@tanstack/react-query";
import type React from "react";
import { useId, useState, useEffect, useRef } from "react";
import { Sliders, ChevronDown, ChevronUp, X } from "lucide-react";
import { createJob, updateJob, describeCron } from "../api/jobs";
import type { Job } from "../types/job";

type JobFormState = {
	id?: string;
	name: string;
	type: Job["type"];
	schedule: string;
	enabled: boolean;
	config: Record<string, string>;
};

type Props = {
	job?: Job;
	onClose: () => void;
};

export default function JobForm({ job, onClose }: Props) {
	const uid = useId();
	const queryClient = useQueryClient();
	const defaultSchedule = job?.schedule ?? "0 0 * * * *";

	const [formData, setFormData] = useState<JobFormState>(
		job
			? {
					id: job.id,
					name: job.name,
					type: job.type,
					schedule: job.schedule,
					enabled: job.enabled,
					config: (job.config || {}) as Record<string, string>,
				}
			: {
					name: "",
					type: "email",
					schedule: defaultSchedule,
					enabled: true,
					config: {} as Record<string, string>,
				},
	);

	const [cronFields, setCronFields] = useState<string[]>(() => {
		const parts = defaultSchedule.split(/\s+/).filter(Boolean);
		const padded = ["0", "*", "*", "*", "*", "*"];
		for (let i = 0; i < Math.min(parts.length, 6); i++) padded[i] = parts[i];
		return padded;
	});

	const [showCronFields, setShowCronFields] = useState(false);

	// Local UI state for email body editor tabs
	const [emailBodyTab, setEmailBodyTab] = useState<"edit" | "preview">("edit");

	const [scheduleDesc, setScheduleDesc] = useState<string>("");
	const [descLoading, setDescLoading] = useState(false);
	const [descError, setDescError] = useState<string | null>(null);
	const debounceRef = useRef<number | null>(null);

	useEffect(() => {
		const composed = cronFields.join(" ");
		setFormData((prev) => ({ ...prev, schedule: composed }));

		if (debounceRef.current) window.clearTimeout(debounceRef.current);
		debounceRef.current = window.setTimeout(async () => {
			setDescLoading(true);
			setDescError(null);
			try {
				const data = await describeCron(composed);
				setScheduleDesc(data.description || "");
			} catch (err) {
				const msg = err instanceof Error ? err.message : String(err);
				setDescError(msg);
				setScheduleDesc("");
			} finally {
				setDescLoading(false);
			}
		}, 600);

		return () => {
			if (debounceRef.current) window.clearTimeout(debounceRef.current);
		};
	}, [cronFields]);

	const createMutation = useMutation({
		mutationFn: (data: Partial<Job>) => createJob(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["jobs"] });
			onClose();
		},
	});

	const updateMutation = useMutation({
		mutationFn: (data: { id: string } & Partial<Job>) => updateJob(data),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["jobs"] });
			onClose();
		},
	});

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		// Convert user's local cron schedule to UTC before sending
		const convertScheduleToUTC = (schedule: string) => {
			const parts = schedule.trim().split(/\s+/);
			if (parts.length !== 6) return schedule;
			const [s, m, h, d, mon, w] = parts;
			const tzOffsetMin = new Date().getTimezoneOffset(); // minutes to add to local to get UTC

			const isNumber = (t: string) => /^\d+$/.test(t);

			// Best-effort: when seconds/minutes/hours are exact numbers, map to UTC by constructing a local Date
			if (isNumber(s) && isNumber(m) && isNumber(h)) {
				const now = new Date();
				const local = new Date(now.getFullYear(), now.getMonth(), now.getDate(), parseInt(h, 10), parseInt(m, 10), parseInt(s, 10));
				const utcS = local.getUTCSeconds();
				const utcM = local.getUTCMinutes();
				const utcH = local.getUTCHours();
				return `${utcS} ${utcM} ${utcH} ${d} ${mon} ${w}`;
			}

			// If timezone offset is whole hours, we can shift hour tokens (best-effort for lists/ranges)
			if (tzOffsetMin % 60 === 0) {
				const hourShift = tzOffsetMin / 60; // may be negative

				const shiftHourToken = (token: string): string => {
					if (/^\d+$/.test(token)) {
						let v = (parseInt(token, 10) + hourShift) % 24;
						if (v < 0) v += 24;
						return String(v);
					}
					if (/^\d+-\d+$/.test(token)) {
						const [a, b] = token.split("-").map(Number);
						let na = (a + hourShift) % 24;
						let nb = (b + hourShift) % 24;
						if (na < 0) na += 24;
						if (nb < 0) nb += 24;
						return `${na}-${nb}`;
					}
					if (/^\d+(,\d+)+$/.test(token)) {
						return token.split(",").map(shiftHourToken).join(",");
					}
					// leave steps/wildcards unchanged
					return token;
				};

				const newH = h.split(",").map(shiftHourToken).join(",");
				return `${s} ${m} ${newH} ${d} ${mon} ${w}`;
			}

			// Fallback: return original schedule when we can't safely convert
			return schedule;
		};

		const utcSchedule = convertScheduleToUTC(formData.schedule);

		if (job) {
			updateMutation.mutate({ ...(formData as Partial<Job>), id: job.id, schedule: utcSchedule });
		} else {
			// Send an explicit empty id so the backend can assign one server-side
			createMutation.mutate({ ...(formData as Partial<Job>), id: "", schedule: utcSchedule });
		}
	};

	const updateConfig = (key: string, value: string) => {
		setFormData((prev) => ({ ...prev, config: { ...prev.config, [key]: value } }));
	};

	const jobTypes = ["email", "sync", "backup", "custom"];

	const renderConfigFields = () => {
		switch (formData.type) {
			case "email":
				return (
					<>
						<input
							type="email"
							placeholder="To"
							value={formData.config.to || ""}
							onChange={(e) => updateConfig("to", e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
						<input
							type="text"
							placeholder="Subject"
							value={formData.config.subject || ""}
							onChange={(e) => updateConfig("subject", e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
							<div className="w-full">
								<div className="flex items-center border-b border-gray-200 mb-2">
									<button
										type="button"
										className={`px-3 py-2 text-sm font-medium ${emailBodyTab === "edit" ? "text-blue-600 border-b-2 border-blue-600" : "text-gray-600"}`}
										onClick={() => setEmailBodyTab("edit")}
									>
										Edit HTML
									</button>
									<button
										type="button"
										className={`ml-2 px-3 py-2 text-sm font-medium ${emailBodyTab === "preview" ? "text-blue-600 border-b-2 border-blue-600" : "text-gray-600"}`}
										onClick={() => setEmailBodyTab("preview")}
									>
										Preview
									</button>
								</div>
								{emailBodyTab === "edit" ? (
									<textarea
										placeholder="Body (HTML)"
										value={formData.config.body || ""}
										onChange={(e) => updateConfig("body", e.target.value)}
										className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
										rows={8}
									/>
								) : (
									<div className="w-full border border-gray-200 rounded-lg bg-gray-50 overflow-hidden" style={{ height: 240 }}>
										<iframe
											title="Email body preview"
											className="w-full h-full"
											sandbox=""
											srcDoc={formData.config.body && formData.config.body.trim().length > 0 ? formData.config.body : "<em style='color:#6b7280;font-family:ui-sans-serif,system-ui;'>No content</em>"}
										/>
									</div>
								)}
								<div className="mt-1 text-xs text-gray-500">Compose HTML and switch to Preview to see rendering.</div>
							</div>
					</>
				);
			case "sync":
				return (
					<>
						<input
							type="text"
							placeholder="Source"
							value={formData.config.source || ""}
							onChange={(e) => updateConfig("source", e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
						<input
							type="text"
							placeholder="Destination"
							value={formData.config.destination || ""}
							onChange={(e) => updateConfig("destination", e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
					</>
				);
			case "backup":
				return (
					<>
						<input
							type="text"
							placeholder="Path"
							value={formData.config.path || ""}
							onChange={(e) => updateConfig("path", e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
						<input
							type="text"
							placeholder="Destination"
							value={formData.config.destination || ""}
							onChange={(e) => updateConfig("destination", e.target.value)}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
					</>
				);
			case "custom":
				return (
					<input
						type="text"
						placeholder="Command"
						value={formData.config.command || ""}
						onChange={(e) => updateConfig("command", e.target.value)}
						className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
						required
					/>
				);
			default:
				return null;
		}
	};

	return (
		<div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
			<div className="bg-white rounded-xl shadow-2xl max-w-md w-full p-6 relative">
				<button type="button" onClick={onClose} aria-label="Close" className="absolute top-3 right-3 text-gray-500 hover:text-gray-800 focus:outline-none">
					<X className="w-5 h-5 icon dark:text-red-500" aria-hidden />
				</button>
				<h2 className="text-2xl font-bold mb-6 text-gray-800">{job ? "Edit Job" : "Create New Job"}</h2>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div>
						<label htmlFor={`${uid}-name`} className="block text-sm font-medium text-gray-700 mb-1">Job Name</label>
						<input
							id={`${uid}-name`}
							type="text"
							value={formData.name}
							onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
							required
						/>
					</div>

					<div>
						<label htmlFor={`${uid}-type`} className="block text-sm font-medium text-gray-700 mb-1">Job Type</label>
						<select
							id={`${uid}-type`}
							value={formData.type}
							onChange={(e) => setFormData((prev) => ({ ...prev, type: e.target.value as Job["type"], config: {} }))}
							className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
						>
							{/* Ensure any unknown type (newly added) shows up in the dropdown */}
							{jobTypes.includes(formData.type) ? null : (
								<option value={formData.type}>{formData.type}</option>
							)}
							{jobTypes.map((t) => (
								<option key={t} value={t}>{t.charAt(0).toUpperCase() + t.slice(1)}</option>
							))}
						</select>
					</div>

					<div>
						<label htmlFor={`${uid}-cron-composed`} className="block text-sm font-medium text-gray-700 mb-2">Schedule</label>
						<div className="flex items-center gap-2">
											<div id={`${uid}-cron-composed`} className="font-mono text-xs text-gray-700">{cronFields.join(" ")}</div>
											<button
												type="button"
												onClick={() => setShowCronFields((s) => !s)}
												className="flex items-center gap-1 text-blue-600 hover:text-blue-800 focus:outline-none"
												aria-expanded={showCronFields}
												aria-controls={`${uid}-cron-detailed`}
											>
												<span className="sr-only">{showCronFields ? "Hide detailed editor" : "Edit detailed schedule"}</span>
												<Sliders className="w-4 h-4 icon dark:text-red-500" aria-hidden />
					{showCronFields ? <ChevronUp className="w-3 h-3 ml-1 icon dark:text-red-500" aria-hidden /> : <ChevronDown className="w-3 h-3 ml-1 icon dark:text-red-500" aria-hidden />}
											</button>
						</div>

						<div className="mt-2 text-sm">
							{descLoading ? (
								<div className="text-xs text-gray-500">Loading description...</div>
							) : descError ? (
								<div className="text-xs text-red-500">{descError}</div>
							) : scheduleDesc ? (
								<div className="text-xs text-gray-600">{scheduleDesc}</div>
							) : (
								<div className="text-xs text-gray-500">Enter cron fields to get a human-readable description.</div>
							)}
						</div>

						{showCronFields && (
							<div className="grid grid-cols-3 gap-2 mt-3">
								{["seconds", "minutes", "hours", "day", "month", "weekday"].map((label, idx) => {
									const id = `${uid}-cron-${idx}`;
									const hints: Record<string, string> = {
										seconds: "0-59 or * or */n",
										minutes: "0-59 or * or */n",
										hours: "0-23 or * or */n",
										day: "1-31 or *",
										month: "1-12 or *",
										weekday: "0-6 (Sun=0) or *",
									};
									return (
										<div key={label} className="flex flex-col">
											<label htmlFor={id} className="text-xs text-gray-500">{label}</label>
											<input id={id} type="text" value={cronFields[idx]} onChange={(e) => {
												const v = e.target.value;
												setCronFields((prev) => {
													const copy = [...prev];
													copy[idx] = v;
													return copy;
												});
											}} className="px-2 py-1 border border-gray-300 rounded font-mono text-sm" aria-label={label} />
											<div className="text-xs text-gray-400 mt-1">{hints[label]}</div>
										</div>
									);
								})}
							</div>
						)}
					</div>

					<div>
						<label htmlFor={`${uid}-config`} className="block text-sm font-medium text-gray-700 mb-2">Configuration</label>
						{renderConfigFields()}
					</div>

					<div className="flex items-center">
						<input id={`${uid}-enabled`} type="checkbox" checked={formData.enabled} onChange={(e) => setFormData((prev) => ({ ...prev, enabled: e.target.checked }))} className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500" />
						<label htmlFor={`${uid}-enabled`} className="ml-2 text-sm text-gray-700">Enable job</label>
					</div>

					<div className="flex gap-3 pt-4">
						<button type="submit" disabled={createMutation.isPending || updateMutation.isPending} className="flex-1 bg-blue-600 text-white py-2 px-4 rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium transition-colors">{createMutation.isPending || updateMutation.isPending ? "Saving..." : job ? "Update Job" : "Create Job"}</button>
						<button type="button" onClick={onClose} className="flex-1 bg-gray-200 text-gray-800 py-2 px-4 rounded-lg hover:bg-gray-300 font-medium transition-colors">Cancel</button>
					</div>
				</form>
			</div>
		</div>
	);
}

                    