const configuredBaseUrl = import.meta.env.VITE_SUB3API_BASE_URL?.trim()

export const apiBaseUrl = (configuredBaseUrl || "http://127.0.0.1:18081").replace(/\/+$/, "")
