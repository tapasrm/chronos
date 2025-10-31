import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Calendar, Clock, Edit2, Power, PowerOff, Trash2 } from "lucide-react";
import { deleteJob, updateJob } from "../api/jobs";
import type { Job } from "../types/job";

type Props = {
	job: Job;
	onEdit: (job: Job) => void;
};

export default function JobCard({ job, onEdit }: Props) {
	const queryClient = useQueryClient();

	const deleteMutation = useMutation({
		mutationFn: deleteJob,
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["jobs"] }),
	});

	const toggleMutation = useMutation({
		mutationFn: updateJob,
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["jobs"] }),
	});

	const handleToggle = () => {
		toggleMutation.mutate({ ...job, enabled: !job.enabled });
	};

	const getTypeColor = (type: string) => {
		const colors: Record<string, string> = {
			email: "bg-blue-100 text-blue-800",
			sync: "bg-green-100 text-green-800",
			backup: "bg-purple-100 text-purple-800",
			custom: "bg-orange-100 text-orange-800",
		};
		return colors[type] || "bg-gray-100 text-gray-800";
	};

	const formatDate = (date?: string | number | null) => {
		if (date === undefined || date === null || date === "") return "Never";

		// Handle numeric timestamps (seconds or milliseconds) and numeric strings
		let dt: Date | null = null;
		if (typeof date === "number") {
			// if it's clearly milliseconds (> 1e12) use as-is, else treat as seconds
			const ms = date > 1e12 ? date : date * 1000;
			dt = new Date(ms);
		} else if (/^\d+$/.test(String(date))) {
			const n = parseInt(String(date), 10);
			const ms = n > 1e12 ? n : n * 1000;
			dt = new Date(ms);
		} else {
			const parsed = new Date(String(date));
			if (!Number.isNaN(parsed.getTime())) dt = parsed;
		}

		if (!dt) return "Never";
		// toLocaleString uses the user's local timezone by default
		return dt.toLocaleString();
	};

	return (
		<div className="bg-white rounded-lg shadow-md p-6 border border-gray-200 hover:shadow-lg transition-shadow">
			<div className="flex items-start justify-between mb-4">
				<div className="flex-1">
					<div className="flex items-center gap-2 mb-2">
						<h3 className="text-xl font-semibold text-gray-800">{job.name}</h3>
						<span
							className={`px-2 py-1 rounded-full text-xs font-medium ${getTypeColor(job.type)}`}
						>
							{job.type}
						</span>
					</div>
					<p className="text-sm text-gray-600 font-mono bg-gray-50 px-2 py-1 rounded inline-block">
						{job.schedule}
					</p>
					{job.scheduleDesc ? (
						<p className="text-sm text-gray-500 mt-1">{job.scheduleDesc}</p>
					) : null}
				</div>

				<div className="flex gap-2">
					<button
						type="button"
						onClick={handleToggle}
						className={`p-2 rounded-lg transition-colors ${job.enabled ? "bg-green-100 text-green-700 hover:bg-green-200" : "bg-gray-100 text-gray-500 hover:bg-gray-200"}`}
						title={job.enabled ? "Disable" : "Enable"}
					>
						{job.enabled ? <Power size={18} className="icon dark:text-red-500" /> : <PowerOff size={18} className="icon dark:text-red-500" />}
					</button>
					<button
						type="button"
						onClick={() => onEdit(job)}
						className="p-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-colors"
						title="Edit"
					>
						<Edit2 size={18} className="icon dark:text-red-500" />
					</button>
					<button
						type="button"
						onClick={() => deleteMutation.mutate(job.id)}
						className="p-2 bg-red-100 text-red-700 rounded-lg hover:bg-red-200 transition-colors"
						title="Delete"
					>
						<Trash2 size={18} className="icon dark:text-red-500" />
					</button>
				</div>
			</div>

			<div className="space-y-2 text-sm">
				<div className="flex items-center gap-2 text-gray-600">
					<Clock size={16} className="icon dark:text-red-500" />
					<span>Last run: {formatDate(job.lastRun)}</span>
				</div>
				<div className="flex items-center gap-2 text-gray-600">
					<Calendar size={16} className="icon dark:text-red-500" />
					<span>Next run: {formatDate(job.nextRun)}</span>
				</div>
			</div>

			<div className="mt-4 pt-4 border-t border-gray-200">
				<p className="text-xs font-medium text-gray-500 mb-2">Configuration:</p>
				<div className="text-sm text-gray-700 space-y-1">
					{Object.entries(job.config || {}).map(([key, value]) => (
						<div key={key} className="flex gap-2">
							<span className="font-medium text-gray-600">{key}:</span>
							<span className="text-gray-800 truncate">{String(value)}</span>
						</div>
					))}
				</div>
			</div>
		</div>
	);
}
