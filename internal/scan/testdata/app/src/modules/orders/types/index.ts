import { z } from 'zod';

export const orderStatusSchema = z.enum([
  'pending',
  // 'on_the_way_to_lmh',
  'received_at_lmh',
  'delivered'
]);

export const returnOrderStatusSchema = z.enum(['paid_return', 'delivered']);

export const orderStatusGroupSchema = z.enum(['active', 'returned']);
export const hubStatusSchema = z.enum(['active', 'disabled']);
