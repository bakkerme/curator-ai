import { NextResponse } from 'next/server';
import { updateSelectors } from '@/lib/runStore';

export async function POST(request: Request, { params }: { params: { id: string } }) {
  const body = (await request.json()) as Record<string, string>;
  const run = await updateSelectors(params.id, body);
  if (!run) {
    return NextResponse.json({ error: 'Run not found or not ready for preview' }, { status: 404 });
  }

  return NextResponse.json(run);
}
