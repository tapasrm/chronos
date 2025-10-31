import { useQuery } from "@tanstack/react-query";
import { Clock, Plus } from "lucide-react";
import { useState } from "react";
import { fetchJobs } from "../api/jobs";
import type { Job } from "../types/job";
import JobCard from "./JobCard";
import JobForm from "./JobForm";

export default function CronJobManager() {
	const [showForm, setShowForm] = useState(false);
	const [editingJob, setEditingJob] = useState<Job | null>(null);

	const {
		data: jobs,
		isLoading,
		error,
	} = useQuery<Job[], Error>({
		queryKey: ["jobs"],
		queryFn: fetchJobs,
		refetchInterval: 5000,
	});

	const handleEdit = (job: Job) => {
		setEditingJob(job);
		setShowForm(true);
	};

	const handleCloseForm = () => {
		setShowForm(false);
		setEditingJob(null);
	};

	if (isLoading) {
		return (
			<div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
				<div className="text-xl text-gray-600">Loading jobs...</div>
			</div>
		);
	}

	if (error) {
		const message = error?.message || "Unknown error";
		return (
			<div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
				<div className="bg-red-50 border border-red-200 text-red-700 px-6 py-4 rounded-lg">
					Error: {message}. Make sure the backend is running on port 8080.
				</div>
			</div>
		);
	}

	return (
		<div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 p-8">
			<div className="max-w-7xl mx-auto">
				<div className="flex items-center justify-between mb-8">
					<div>
						<h1 className="text-4xl font-bold text-gray-800 mb-2">
							Cron Job Manager
						</h1>
						<p className="text-gray-600">
							Manage your scheduled tasks and automation
						</p>
					</div>
					<button
						type="button"
						onClick={() => setShowForm(true)}
						className="flex items-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700 transition-colors shadow-lg font-medium"
					>
						<Plus size={20} className="icon dark:text-red-500" />
						New Job
					</button>
				</div>

				{jobs && jobs.length === 0 ? (
					<div className="bg-white rounded-xl shadow-md p-12 text-center">
						<div className="text-gray-400 mb-4">
							<Clock size={64} className="mx-auto icon dark:text-red-500" />
						</div>
						<h3 className="text-xl font-semibold text-gray-700 mb-2">
							No jobs configured
						</h3>
						<p className="text-gray-500 mb-6">
							Get started by creating your first cron job
						</p>
						<button
							type="button"
							onClick={() => setShowForm(true)}
							className="inline-flex items-center gap-2 bg-blue-600 text-white px-6 py-3 rounded-lg hover:bg-blue-700 transition-colors font-medium"
						>
								<Plus size={20} className="icon dark:text-red-500" />
							Create First Job
						</button>
					</div>
				) : (
					<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
						{jobs?.map((job) => (
							<JobCard key={job.id} job={job} onEdit={handleEdit} />
						))}
					</div>
				)}

				{showForm && (
					<JobForm job={editingJob ?? undefined} onClose={handleCloseForm} />
				)}
			</div>
		</div>
	);
}
