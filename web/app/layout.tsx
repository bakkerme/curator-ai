import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Curator Web (v0)',
  description: 'Scrape block generator vertical slice'
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
