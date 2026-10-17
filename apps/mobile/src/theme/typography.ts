import { TextStyle } from "react-native";
import { colors } from "./tokens";

export const fonts = {
  headline: "Syne_700Bold",
  headlineRegular: "Syne_400Regular",
  body: "IBMPlexSans_400Regular",
  bodyMedium: "IBMPlexSans_500Medium",
  label: "SpaceGrotesk_500Medium",
  labelBold: "SpaceGrotesk_700Bold",
} as const;

export const typography = {
  brand: {
    fontFamily: fonts.headline,
    fontSize: 28,
    letterSpacing: 1,
    color: colors.onSurface,
  } satisfies TextStyle,
  title: {
    fontFamily: fonts.headline,
    fontSize: 22,
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
    letterSpacing: 1.2,
    textTransform: "uppercase",
    color: colors.tertiary,
  } satisfies TextStyle,
  caption: {
    fontFamily: fonts.body,
    fontSize: 13,
    lineHeight: 18,
    color: colors.onSurfaceMuted,
  } satisfies TextStyle,
} as const;
