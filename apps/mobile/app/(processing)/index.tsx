import { ProcessingScreen } from "@/features/moment/ProcessingScreen";

/** Cut draft + Render wait. Dismissible — poller survives minimize (ADR 0018). */
export default function ProcessingRoute() {
  return <ProcessingScreen />;
}
