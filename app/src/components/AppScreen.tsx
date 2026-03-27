import { PropsWithChildren } from 'react';
import { SafeAreaView, StyleSheet, View } from 'react-native';

type AppScreenProps = PropsWithChildren<{
  padded?: boolean;
}>;

export function AppScreen({ children, padded = true }: AppScreenProps) {
  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={[styles.content, padded && styles.padded]}>{children}</View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: '#f7f2e8',
  },
  content: {
    flex: 1,
  },
  padded: {
    paddingHorizontal: 24,
    paddingVertical: 20,
  },
});
