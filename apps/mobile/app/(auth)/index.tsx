import { useState } from "react";
import { Alert, StyleSheet, Text, View } from "react-native";
import { Redirect } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandMark } from "@/components/BrandMark";
import { BrassButton } from "@/components/BrassButton";
import { useAuth } from "@/lib/auth";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";

/** Minimal auth gate (not in Stitch set) — anonymous OK per GRILL Q2. */
export default function AuthScreen() {
  const { session, loading, configured, signInAnonymously } = useAuth();
  const [busy, setBusy] = useState(false);

  if (!loading && session) {
    return <Redirect href="/(onboarding)" />;
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
        <Text style={styles.title}>Sign in to continue</Text>
        <Text style={styles.body}>
          Week Intake and gateway calls need a session. Anonymous sign-in is fine
          for this MVP.
        </Text>
        {!configured ? (
          <Text style={styles.warn}>
            Set EXPO_PUBLIC_SUPABASE_URL and EXPO_PUBLIC_SUPABASE_ANON_KEY before
            signing in.
          </Text>
        ) : null}
        <BrassButton
          label="Continue anonymously"
          onPress={() => void continueAnon()}
          loading={busy || loading}
          disabled={!configured}
        />
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
