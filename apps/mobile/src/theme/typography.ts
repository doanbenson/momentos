import { TextStyle } from "react-native";
import { colors } from "./tokens";

/** Obsidian Sunset: Plus Jakarta Sans + Space Grotesk. */
export const fonts = {
  headline: "PlusJakartaSans_700Bold",
  headlineRegular: "PlusJakartaSans_600SemiBold",
  body: "PlusJakartaSans_400Regular",
  bodyMedium: "PlusJakartaSans_500Medium",
  label: "SpaceGrotesk_500Medium",
  labelBold: "SpaceGrotesk_700Bold",
} as const;

export const typography = {
  brand: {
    fontFamily: fonts.headline,
    fontSize: 28,
    letterSpacing: -0.5,
    color: colors.onSurface,
  } satisfies TextStyle,
  title: {
    fontFamily: fonts.headlineRegular,
    fontSize: 22,
    letterSpacing: -0.2,
    color: colors.onSurface,
  } satisfies TextStyle,
  body: {
    fontFamily: fonts.body,
    fontSize: 16,
    lineHeight: 24,
    color: colors.onSurfaceMuted,
  } satisfies TextStyle,
  bodyStrong: {
    fontFamily: fonts.bodyMedium,
    fontSize: 16,
    lineHeight: 24,
    color: colors.onSurface,
  } satisfies TextStyle,
  label: {
    fontFamily: fonts.label,
    fontSize: 12,
    letterSpacing: 0.5,
    color: colors.onSurfaceMuted,
  } satisfies TextStyle,
  labelStrong: {
    fontFamily: fonts.labelBold,
    fontSize: 12,
    letterSpacing: 0.5,
    color: colors.onSurface,
  } satisfies TextStyle,
  caption: {
    fontFamily: fonts.body,
    fontSize: 12,
    lineHeight: 16,
    color: colors.onSurfaceMuted,
  } satisfies TextStyle,
} as const;
