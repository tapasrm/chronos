export type Job = {
	id: string;
	name: string;
	type: "email" | "sync" | "backup" | "custom" | string;
	schedule: string;
	scheduleDesc: string;
	enabled: boolean;
	lastRun?: string | null;
	nextRun?: string | null;
	config?: Record<string, string>;
};
