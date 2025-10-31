import type { Job } from "../types/job";

// Use relative URL in development (proxied by Vite) or when served from same origin
// Fall back to absolute URL for production when frontend and backend are separate
export const API_BASE = "/api";

export const fetchJobs = async (): Promise<Job[]> => {
  const res = await fetch(`${API_BASE}/jobs`);
  if (!res.ok) throw new Error("Failed to fetch jobs");
  return res.json();
};

export const createJob = async (job: Partial<Job>) => {
  const res = await fetch(`${API_BASE}/jobs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(job),
  });
  if (!res.ok) throw new Error("Failed to create job");
  return res.json();
};

export const findJobByName = async (name: string) => {
  const res = await fetch(`${API_BASE}/jobs?name=${name}`);
  if (!res.ok) throw new Error("Failed to find job");
  return res.json();
};

export const updateJob = async ({
  id,
  ...job
}: { id: string } & Partial<Job>) => {
  const res = await fetch(`${API_BASE}/jobs/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(job),
  });
  if (!res.ok) throw new Error("Failed to update job");
  return res.json();
};

export const deleteJob = async (id: string) => {
  const res = await fetch(`${API_BASE}/jobs/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete job");
};

export const describeCron = async (schedule: string): Promise<{ description: string }> => {
  const res = await fetch(`${API_BASE}/describe-cron`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ schedule }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  return res.json();
};
