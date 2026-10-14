import { api } from '$lib/api';
import { redirect } from '@sveltejs/kit';

export async function load() {
  try {
    return { user: await api('/session', { redirectOnUnauthorized: false }) };
  } catch (error) {
    if (error.status === 401) redirect(303, '/login');
    throw error;
  }
}
