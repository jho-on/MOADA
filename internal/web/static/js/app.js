// MOADA frontend bootstrap.
//
// The application uses server-rendered Templ pages updated through HTMX. This
// module owns the shared browser state that must survive partial navigation.
// Cryptographic keys must never be exposed on the window object or in the DOM.

import { apiFetch, ApiError } from "./modules/api.js";

const MOADA = {
	version: "0.1.0",
	api: {
		fetch: apiFetch,
		ApiError,
	},
};

function setConnectionState(state) {
	document.documentElement.dataset.connection = state;
}

document.addEventListener("htmx:beforeRequest", () => {
	setConnectionState("loading");
});

document.addEventListener("htmx:afterRequest", (event) => {
	setConnectionState(event.detail.successful ? "idle" : "error");
});

document.addEventListener("htmx:sendError", () => {
	setConnectionState("offline");
});

document.addEventListener("htmx:responseError", () => {
	setConnectionState("error");
});

export { MOADA };
