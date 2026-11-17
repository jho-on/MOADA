// Small wrapper around fetch for JSON API calls made by the web interface.
// It centralizes credentials, headers and error handling so feature modules do
// not duplicate those details.

export class ApiError extends Error {
	constructor(message, { status = 0, body = null } = {}) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.body = body;
	}
}

export async function apiFetch(path, options = {}) {
	const headers = new Headers(options.headers);
	if (!headers.has("Accept")) {
		headers.set("Accept", "application/json");
	}

	const response = await fetch(path, {
		...options,
		headers,
		credentials: "same-origin",
	});

	if (!response.ok) {
		const contentType = response.headers.get("Content-Type") || "";
		let body = null;

		if (contentType.includes("application/json")) {
			body = await response.json().catch(() => null);
		} else {
			body = await response.text().catch(() => null);
		}

		const message =
			body && typeof body === "object" && typeof body.error === "string"
				? body.error
				: `Request failed with status ${response.status}`;

		throw new ApiError(message, { status: response.status, body });
	}

	if (response.status === 204) {
		return null;
	}

	const contentType = response.headers.get("Content-Type") || "";
	if (contentType.includes("application/json")) {
		return response.json();
	}
	return response.text();
}
