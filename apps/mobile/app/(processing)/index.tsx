import { ProcessingScreen } from "@/features/moment/ProcessingScreen";

/** Waiting on Cut draft (ml) + Render. Non-dismissible until ready/failed. */
export default function ProcessingRoute() {
  return <ProcessingScreen />;
}
