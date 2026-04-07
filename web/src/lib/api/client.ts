import createClient from 'openapi-fetch';
import type { paths } from './types.gen';

import { API_BASE_URL } from '$env/static/private';

export const api = createClient<paths>({ baseUrl: API_BASE_URL });
