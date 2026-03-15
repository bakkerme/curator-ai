import UrlIntakeForm from '@/components/UrlIntakeForm';

export default function NewScrapePage() {
  return (
    <main className="grid">
      <h1>Scrape Block Generator</h1>
      <p>Provide a target URL and run selector discovery.</p>
      <UrlIntakeForm />
    </main>
  );
}
