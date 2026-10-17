import { Text, View } from "react-native";

/** Week Intake — up to 10 Assets (photo / video / voice). */
export default function OnboardingScreen() {
  return (
    <View style={{ flex: 1, justifyContent: "center", padding: 24 }}>
      <Text style={{ fontSize: 24, fontWeight: "600" }}>Week Intake</Text>
      <Text style={{ marginTop: 8 }}>
        Give us up to 10 photos, videos, or voice recordings from your past week.
      </Text>
    </View>
  );
}
