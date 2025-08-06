// App.tsx
import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import WelcomeScreen from './screens/WelcomeScreen';
import SelectSpreadScreen from './screens/SelectSpreadScreen';
import ResultScreen from './screens/ResultScreen';
import HistoryScreen from './screens/HistoryScreen';
import { useColorScheme, View, Text, Image, TouchableOpacity } from 'react-native';

export type RootStackParamList = {
  Welcome: undefined;
  SelectSpread: undefined;
  Result: { cards: TarotCard[] };
  History: undefined;
};

export type TarotCard = {
  name: string;
  upright: string;
  reversed: string;
  image: string;
  isReversed: boolean;
};

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function App() {
  const scheme = useColorScheme(); // "dark" | "light"

  return (
      <NavigationContainer>
        <Stack.Navigator initialRouteName="Welcome">
          <Stack.Screen
              name="Welcome"
              options={{ headerShown: false }}
          >
            {props => <WelcomeScreen {...props} theme={scheme} />}
          </Stack.Screen>

          <Stack.Screen
              name="SelectSpread"
              options={{
                title: "📜 Выбор расклада",
                headerShown: true,
                headerTitleAlign: 'center',
                headerStyle: {
                  backgroundColor: scheme === 'dark' ? '#111' : '#fff',
                },
                headerTintColor: scheme === 'dark' ? '#fff' : '#000',
              }}
          >
            {props => <SelectSpreadScreen {...props} theme={scheme} />}
          </Stack.Screen>

          <Stack.Screen
              name="Result"
              options={({ navigation }) => ({
                headerTitle: () => (
                    <View style={{ flexDirection: 'row', alignItems: 'center' }}>
                      <Image
                          source={require('./assets/logo.png')} // ❗ Замени на свой логотип
                          style={{ width: 30, height: 30, marginRight: 8 }}
                      />
                      <Text style={{ fontSize: 18, fontWeight: 'bold', color: scheme === 'dark' ? '#fff' : '#000' }}>
                        🔮 Результат
                      </Text>
                    </View>
                ),
                headerRight: () => (
                    <TouchableOpacity onPress={() => navigation.navigate('History')}>
                      <Text style={{ marginRight: 12, fontSize: 16, color: scheme === 'dark' ? '#fff' : '#000' }}>
                        🕓
                      </Text>
                    </TouchableOpacity>
                ),
                headerStyle: {
                  backgroundColor: scheme === 'dark' ? '#111' : '#fff',
                },
                headerTintColor: scheme === 'dark' ? '#fff' : '#000',
              })}
          >
            {props => <ResultScreen {...props} theme={scheme} />}
          </Stack.Screen>

          <Stack.Screen
              name="History"
              options={{
                title: "🕓 История",
                headerShown: true,
                headerTitleAlign: 'center',
                headerStyle: {
                  backgroundColor: scheme === 'dark' ? '#111' : '#fff',
                },
                headerTintColor: scheme === 'dark' ? '#fff' : '#000',
              }}
          >
            {props => <HistoryScreen {...props} theme={scheme} />}
          </Stack.Screen>
        </Stack.Navigator>
      </NavigationContainer>
  );
}
