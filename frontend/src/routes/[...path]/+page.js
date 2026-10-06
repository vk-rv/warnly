import { api } from '$lib/api';

export async function load({ url }) {
  const path = url.pathname.replace(/\/$/, '') || '/';
  const placeholder = ['/oncall', '/analytics', '/notready', '/error'].includes(path);
  try {
    const user = path === '/login' ? null : await api('/session');
    const result = placeholder ? null : await api((path === '/' ? '/issues' : path) + url.search);
    const issuePath = /^\/projects\/\d+\/issues\/\d+/.exec(path)?.[0];
    const issue = issuePath && path !== issuePath ? await api(issuePath) : null;
    return { path, result, issue, user, error: '' };
  } catch (error) {
    return { path, result: null, user: null, error: error.message };
  }
}
