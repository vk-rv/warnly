import { api } from '$lib/api';

export async function load() {
  try {
    return { path: '/login', result: await api('/login'), error: '' };
  } catch (error) {
    return { path: '/login', result: null, error: error.message };
  }
}
