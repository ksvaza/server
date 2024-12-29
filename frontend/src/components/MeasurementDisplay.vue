<script setup lang="ts">
  import { ref } from 'vue';
  import MeasurementDisplay from './components/MeasurementDisplay.vue';

  export default {
  name: 'MeasurementDisplay',
  props: {
    timeStamp: String,
    temperature: String,
    humidity: String,
    battery: String,
    rssi: String
  }
}

  const timeStamp = ref"2024";
  const temperature = ref"39.6";
  const humidity = ref"50.0";
  const battery = ref"0.22";
  const rssi = ref"88";
</script>

<template>
    <div class="measurementDisplay">
        <table>
            <tr>
                <th>Timestamp</th> <th>Temperature</th> <th>Humidity</th> <th>Battery</th> <th>RSSI</th>
            </tr>
            <tr>
                <td>{{ timeStamp }}</td> <td>{{ temperature }}</td> <td>{{ humidity }}</td> <td>{{ battery }}</td> <td>{{ rssi }}</td>
            </tr>
        </table>
    </div>
</template>

<style>
.measurementDisplay {
    display: box;
    text-align: center;
    justify-content: center;
    flex-direction: column;
    background-color: #d9d9d9;
    padding: 8px;
    border-radius: 8px;
    border: 2px solid #000000;
    margin: 4px;
    width: fit-content;
}
</style>