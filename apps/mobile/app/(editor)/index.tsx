import { Text, View } from "react-native";

/** MVP Editor: preview, reorder, replace, remove, re-render. */
export default function EditorScreen() {
  return (
    <View style={{ flex: 1, justifyContent: "center", padding: 24 }}>
      <Text style={{ fontSize: 24, fontWeight: "600" }}>Editor</Text>
      <Text style={{ marginTop: 8 }}>
        Preview your Moment. Reorder, replace, or remove Assets, then re-render.
      </Text>
    </View>
  );
}
