import { Text, View } from "react-native";

/** Waiting on Cut draft (ml) + Render. */
export default function ProcessingScreen() {
  return (
    <View style={{ flex: 1, justifyContent: "center", padding: 24 }}>
      <Text style={{ fontSize: 24, fontWeight: "600" }}>Crafting your Moment</Text>
      <Text style={{ marginTop: 8 }}>Drafting a Cut and rendering…</Text>
    </View>
  );
}
