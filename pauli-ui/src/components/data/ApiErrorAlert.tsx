import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { PauliApiError } from "@/lib/api/fetch-json";

export function ApiErrorAlert({ error }: { error: unknown }) {
  const message =
    error instanceof PauliApiError
      ? error.message
      : error instanceof Error
        ? error.message
        : "Something went wrong";

  return (
    <Alert variant="destructive" className="border-destructive/50 bg-destructive/10">
      <AlertTitle>Request failed</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}
