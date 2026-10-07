import restApiClient from '@/lib/api-client';
import { API_ROUTES } from '@/lib/routes/api-routes';
import { Basket } from '@/modules/basket';
// import { Ghost } from '../../ghost';
/* import { Old } from '@/modules/old'; */
export const ordersApi = { list: () => restApiClient.get(API_ROUTES.orders.list), csv: () => API_ROUTES.bulkOrderCsv };
