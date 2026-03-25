import { NextResponse } from 'next/server';
import { getRun } from '@/lib/runStore';

export async function GET(_: Request, { params }: { params: { id: string } }) {
  const run = await getRun(params.id);
  if (!run) {
    return NextResponse.json({ error: 'Run not found' }, { status: 404 });
  }
  return NextResponse.json(run);
}
