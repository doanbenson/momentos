import "react-native-gesture-handler";
import { useEffect } from "react";
import { ActivityIndicator, View } from "react-native";
import { Stack } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useFonts } from "expo-font";
import {
  PlusJakartaSans_400Regular,
  PlusJakartaSans_500Medium,
  PlusJakartaSans_600SemiBold,
  PlusJakartaSans_700Bold,
} from "@expo-google-fonts/plus-jakarta-sans";
import {
  SpaceGrotesk_500Medium,
  SpaceGrotesk_700Bold,
} from "@expo-google-fonts/space-grotesk";
import { CompositionStatusPill } from "@/features/moment/CompositionStatusPill";
import { MomentJobsProvider } from "@/features/moment/MomentJobsProvider";
import { ReadyToast } from "@/features/moment/ReadyToast";
import { AuthProvider } from "@/lib/auth";
import { colors } from "@/theme/tokens";

export default function RootLayout() {
  const [fontsLoaded, fontError] = useFonts({
    PlusJakartaSans_400Regular,
    PlusJakartaSans_500Medium,
    PlusJakartaSans_600SemiBold,
    PlusJakartaSans_700Bold,
    SpaceGrotesk_500Medium,
    SpaceGrotesk_700Bold,
  });

  useEffect(() => {
    if (fontError) {
      console.warn("Font load error", fontError);
    }
  }, [fontError]);

  if (!fontsLoaded && !fontError) {
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

  return (
    <AuthProvider>
      <MomentJobsProvider>
        <StatusBar style="light" />
        <View style={{ flex: 1, backgroundColor: colors.background }}>
          <Stack
            screenOptions={{
              headerShown: false,
              contentStyle: { backgroundColor: colors.background },
              animation: "slide_from_right",
            }}
          >
            <Stack.Screen name="index" />
            <Stack.Screen name="(auth)" />
            <Stack.Screen name="(home)" />
            <Stack.Screen name="(onboarding)" />
            <Stack.Screen name="(processing)" />
            <Stack.Screen name="(editor)" />
            <Stack.Screen name="(share)" />
          </Stack>
          <CompositionStatusPill />
          <ReadyToast />
        </View>
      </MomentJobsProvider>
    </AuthProvider>
  );
}
