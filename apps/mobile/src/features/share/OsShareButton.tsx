import { Alert, Share } from "react-native";
import * as Sharing from "expo-sharing";
import { BrassButton } from "@/components/BrassButton";

type Props = {
  renderUrl?: string;
  title?: string;
};

export function OsShareButton({ renderUrl, title }: Props) {
  async function onShare() {
    try {
      if (renderUrl && (await Sharing.isAvailableAsync())) {
        // Prefer file share when we have a local/remote URI Sharing can open
        try {
          await Sharing.shareAsync(renderUrl, {
            dialogTitle: title ?? "Share Moment",
            mimeType: "video/mp4",
          });
          return;
        } catch {
          // fall through to Share.share
        }
      }
      await Share.share({
        message: renderUrl
          ? `My Momentos Moment\n${renderUrl}`
          : "My Momentos Moment",
        url: renderUrl,
        title: title ?? "Share Moment",
      });
    } catch (e) {
      Alert.alert("Share failed", e instanceof Error ? e.message : String(e));
    }
  }

  return <BrassButton label="Share…" onPress={() => void onShare()} />;
}
