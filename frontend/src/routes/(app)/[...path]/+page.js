import { api } from '$lib/api';
import { redirect } from '@sveltejs/kit';

const loadAPI = (path) => api(path, { redirectOnUnauthorized: false });

export async function load({ url, depends }) {
  depends('app:page');
  const path = url.pathname.replace(/\/$/, '') || '/';
  const placeholder = ['/oncall', '/analytics', '/notready', '/error'].includes(path);
  try {
    const result = placeholder
      ? null
      : await loadAPI((path === '/' ? '/issues' : path) + url.search);
    const issuePath = /^\/projects\/\d+\/issues\/\d+/.exec(path)?.[0];
    const issue = issuePath && path !== issuePath ? await loadAPI(issuePath) : null;
    return { path, result, issue, error: '' };
  } catch (error) {
    if (error.status === 401) redirect(303, '/login');
    return { path, result: null, error: error.message };
  }
}
