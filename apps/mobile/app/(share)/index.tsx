import { Text, View } from "react-native";

/** Save in-app or share via OS sheet + Instagram Audio Match checklist. */
export default function ShareScreen() {
  return (
    <View style={{ flex: 1, justifyContent: "center", padding: 24 }}>
      <Text style={{ fontSize: 24, fontWeight: "600" }}>Save or share</Text>
      <Text style={{ marginTop: 8 }}>
        Keep your Moment in Momentos, or share the Render and attach the Platform
        Song in Instagram Reels (Audio Match).
      </Text>
    </View>
  );
}
