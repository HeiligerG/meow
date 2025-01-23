### **Überblick über das Monitoring-System "miau"**
1. **Zweck**:
    - Das System überwacht Endpunkte (z. B. Webseiten oder APIs) und prüft, ob sie online und erreichbar sind.
    - Es besteht aus mehreren Komponenten, die verteilt arbeiten können, was es zu einer Cloud-fähigen Anwendung macht.

2. **Repository**:
    - Das Repository heißt "Monitoring Endpoints on the Web" (kurz "meow").
    - Es enthält mehrere ausführbare Programme (Komponenten), die für das Monitoring-System benötigt werden.

---

### **Komponenten des Systems**
1. **Canary CMD**:
    - Ein einfacher Server, der einen Canary-Endpunkt bereitstellt.
    - Dieser Endpunkt gibt `OK` zurück, solange der Server online ist.
    - Wird verwendet, um das Monitoring von Offline-Szenarien zu simulieren.

2. **Config CMD**:
    - Verwaltet die Konfiguration der zu überwachenden Endpunkte.
    - Die Konfiguration wird in einer CSV-Datei (`config.csv`) gespeichert.
    - Unterstützt HTTP-Endpunkte, um die konfigurierten Endpunkte abzurufen oder zu aktualisieren.

3. **Probe CMD**:
    - Führt das eigentliche Monitoring durch.
    - Ruft regelmäßig die Endpunkte auf und protokolliert die Ergebnisse.
    - Bei Fehlern wird ein Alarm ausgelöst.

---

### **Endpunkt-Datenstruktur**
Ein Endpunkt wird durch folgende Attribute definiert:
- **Identifier**: Ein eindeutiger Name für den Endpunkt.
- **URL**: Die Adresse des Endpunkts.
- **Methode**: Die HTTP-Methode (z. B. GET).
- **Erwarteter Status**: Der erwartete HTTP-Statuscode (z. B. 200).
- **Frequenz**: Wie oft der Endpunkt überprüft wird (z. B. alle 5 Minuten).
- **Fail After**: Nach wie vielen fehlgeschlagenen Versuchen ein Alarm ausgelöst wird.

---

### **Schritte zur Einrichtung**
1. **Repository klonen**:
    - Das Repository wird geklont und in der Entwicklungsumgebung geöffnet.

2. **Canary-Server starten**:
    - Der Canary-Server wird mit `go run canaryCmd/canary.go` gestartet.
    - Er läuft auf `localhost:9000` und kann mit `curl` oder im Browser getestet werden.

3. **Config-Server starten**:
    - Die Konfigurationsdatei (`config.csv`) wird angepasst, um die zu überwachenden Endpunkte zu definieren.
    - Der Config-Server wird mit `go run configCmd/config.go -file config.csv` gestartet.
    - Er läuft auf `localhost:8000` und stellt Endpunkte bereit, um die Konfiguration abzurufen oder zu aktualisieren.

4. **Probe-Server starten**:
    - Der Probe-Server wird mit `go run probeCmd/probe.go` gestartet.
    - Er liest die Konfiguration vom Config-Server und beginnt mit dem Monitoring.
    - Die Ergebnisse werden in einer Log-Datei gespeichert.

---

### **Beispiel: Monitoring eines Endpunkts**
1. **Canary-Endpunkt testen**:
    - Der Canary-Server wird gestartet und mit `curl http://localhost:9000/canary` getestet.
    - Wenn der Server online ist, gibt er `OK` zurück.

2. **Endpunkt zur Konfiguration hinzufügen**:
    - Ein neuer Endpunkt wird in der `config.csv`-Datei definiert.
    - Der Config-Server wird neu gestartet, um die Änderungen zu übernehmen.

3. **Monitoring starten**:
    - Der Probe-Server wird gestartet und überwacht die Endpunkte.
    - Wenn ein Endpunkt offline geht, wird ein Alarm ausgelöst.

---

### **Verteilung und Cloud-Fähigkeit**
- Das System ist verteilt und kann auf mehrere Server oder Cloud-Umgebungen aufgeteilt werden.
- Die Konfiguration kann lokal laufen, während das Monitoring in der Cloud durchgeführt wird.
- Dies ermöglicht eine flexible und skalierbare Überwachung von Endpunkten weltweit.

---

### **Zusammenfassung**
- Das Monitoring-System "miau" ist eine verteilte Anwendung, die Endpunkte überwacht und bei Problemen Alarme auslöst.
- Es besteht aus drei Hauptkomponenten: Canary, Config und Probe.
- Die Konfiguration wird in einer CSV-Datei gespeichert und über HTTP-Endpunkte verwaltet.
- Das System ist Cloud-fähig und kann für die Überwachung von Cloud-Anwendungen verwendet werden.