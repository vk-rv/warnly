export async function api(path, options = {}) {
  const { redirectOnUnauthorized = true, ...fetchOptions } = options;
  const response = await fetch(`/api${path}`, { credentials: 'same-origin', ...fetchOptions });
  const text = await response.text();
  let data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    throw new Error('Invalid server response');
  }
  if (!response.ok) {
    if (redirectOnUnauthorized && response.status === 401 && path !== '/login')
      window.location.assign('/login');
    throw Object.assign(new Error(data?.error || `Request failed (${response.status})`), {
      status: response.status
    });
  }
  return data;
}

export function formBody(form) {
  return new URLSearchParams(new FormData(form));
}
