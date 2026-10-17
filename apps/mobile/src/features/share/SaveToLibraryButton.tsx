import { Alert } from "react-native";
import * as FileSystem from "expo-file-system";
import * as MediaLibrary from "expo-media-library";
import { GhostButton } from "@/components/GhostButton";

type Props = {
  renderUrl?: string;
};

export function SaveToLibraryButton({ renderUrl }: Props) {
  async function onSave() {
    if (!renderUrl) {
      Alert.alert("Save", "No Render URL on this Moment yet.");
      return;
    }
    try {
      const perm = await MediaLibrary.requestPermissionsAsync();
      if (!perm.granted) {
        Alert.alert("Photos", "Permission required to save to your library.");
        return;
      }
      const cacheDir = FileSystem.cacheDirectory;
      if (!cacheDir) {
        throw new Error("cacheDirectory unavailable");
      }
      const target = `${cacheDir}momentos-render-${Date.now()}.mp4`;
      const downloaded = await FileSystem.downloadAsync(renderUrl, target);
      await MediaLibrary.saveToLibraryAsync(downloaded.uri);
      Alert.alert("Saved", "Moment saved to your library.");
    } catch (e) {
      Alert.alert("Save failed", e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <GhostButton label="Save to library" onPress={() => void onSave()} />
  );
}
