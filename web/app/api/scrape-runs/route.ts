import { NextResponse } from 'next/server';
import { createRun } from '@/lib/runStore';
import { scrapeRunInputSchema } from '@/lib/schemas';

export async function POST(request: Request) {
  const body = await request.json();
  const parsed = scrapeRunInputSchema.safeParse(body);

  if (!parsed.success) {
    return NextResponse.json({ error: parsed.error.flatten() }, { status: 400 });
  }

  const run = await createRun(parsed.data);
  return NextResponse.json(run, { status: 201 });
}
