import { ActivityIndicator, View } from "react-native";
import { Redirect } from "expo-router";
import { WeekIntakeScreen } from "@/features/intake/WeekIntakeScreen";
import { useAuth } from "@/lib/auth";
import { colors } from "@/theme/tokens";

/** Week Intake — up to 10 Assets (photo / video / voice). */
export default function OnboardingScreen() {
  const { session, loading } = useAuth();

  if (loading) {
    return (
      <View
        style={{
          flex: 1,
          backgroundColor: colors.background,
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <ActivityIndicator color={colors.primary} />
      </View>
    );
  }

  if (!session) {
    return <Redirect href="/(auth)" />;
  }

  return <WeekIntakeScreen />;
}
