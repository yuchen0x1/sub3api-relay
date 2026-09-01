const configuredBaseUrl = import.meta.env.VITE_SUB3API_BASE_URL?.trim()

export const apiBaseUrl = (configuredBaseUrl || "http://localhost:3000").replace(/\/+$/, "")
