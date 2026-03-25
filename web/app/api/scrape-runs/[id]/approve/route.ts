import { NextResponse } from 'next/server';
import { approveRun } from '@/lib/runStore';

export async function POST(_: Request, { params }: { params: { id: string } }) {
  const run = await approveRun(params.id);
  if (!run) {
    return NextResponse.json({ error: 'Run not found' }, { status: 404 });
  }
  return NextResponse.json(run);
}
