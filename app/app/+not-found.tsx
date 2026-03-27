import { Link, Stack } from 'expo-router';
import { StyleSheet, Text, View } from 'react-native';

export default function NotFoundScreen() {
  return (
    <>
      <Stack.Screen options={{ title: 'Not Found' }} />
      <View style={styles.container}>
        <Text style={styles.title}>This route does not exist yet.</Text>

        <Link href="/" style={styles.link}>
          <Text style={styles.linkText}>Return to the FreshCycle home shell</Text>
        </Link>
      </View>
    </>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 20,
    backgroundColor: '#f7f2e8',
  },
  title: {
    fontSize: 20,
    fontWeight: '600',
    color: '#1f3527',
  },
  link: {
    marginTop: 15,
    paddingVertical: 15,
  },
  linkText: {
    fontSize: 14,
    color: '#5d7a63',
  },
});
