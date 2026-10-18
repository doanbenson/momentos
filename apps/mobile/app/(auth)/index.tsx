import { useState } from "react";
import { Alert, Linking, StyleSheet, Text, View } from "react-native";
import { Redirect } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandMark } from "@/components/BrandMark";
import { BrassButton } from "@/components/BrassButton";
import { GhostButton } from "@/components/GhostButton";
import { useAuth } from "@/lib/auth";
import { supabaseProjectRef } from "@/lib/supabase";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";

const providersUrl = supabaseProjectRef
  ? `https://supabase.com/dashboard/project/${supabaseProjectRef}/auth/providers`
  : "https://supabase.com/dashboard";

/** MVP auth gate — anonymous only (OAuth out of scope). */
export default function AuthScreen() {
  const { session, loading, configured, error, signInAnonymously } = useAuth();
  const [busy, setBusy] = useState(false);

  if (!loading && session) {
    return <Redirect href="/(home)" />;
  }

  async function continueAnon() {
    setBusy(true);
    try {
      await signInAnonymously();
    } catch (e) {
      Alert.alert(
        "Sign-in failed",
        e instanceof Error ? e.message : String(e),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={styles.safe}>
      <View style={styles.content}>
        <BrandMark variant="hero" />
        <Text style={styles.title}>Continue to Momentos</Text>
        <Text style={styles.body}>
          MVP uses anonymous Supabase sessions only. No email or OAuth.
        </Text>
        {!configured ? (
          <Text style={styles.warn}>
            Set EXPO_PUBLIC_SUPABASE_URL and EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY before
            continuing.
          </Text>
        ) : null}
        {error ? <Text style={styles.warn}>{error}</Text> : null}
        <BrassButton
          label="Continue anonymously"
          onPress={() => void continueAnon()}
          loading={busy || loading}
          disabled={!configured}
        />
        {error?.includes("Anonymous sign-ins are disabled") ? (
          <GhostButton
            label="Open Supabase providers"
            onPress={() => void Linking.openURL(providersUrl)}
          />
        ) : null}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  content: {
    flex: 1,
    padding: space.lg,
    justifyContent: "center",
    gap: space.md,
  },
  title: { ...typography.title, marginTop: space.lg },
  body: { ...typography.body },
  warn: { ...typography.caption, color: colors.error },
});
