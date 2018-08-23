package ch.epfl.prifiproxy.persistence.entity;

import android.arch.persistence.room.Entity;
import android.arch.persistence.room.Index;
import android.arch.persistence.room.PrimaryKey;
import android.support.annotation.NonNull;

@Entity(indices = {@Index(value = {"name"}, unique = true)})
public class ConfigurationGroup {
    @PrimaryKey(autoGenerate = true)
    private int id;

    @NonNull
    private String name;

    private boolean isActive;

    public ConfigurationGroup(int id, @NonNull String name, boolean isActive) {
        this.id = id;
        this.name = name;
        this.isActive = isActive;
    }

    public int getId() {
        return id;
    }

    public void setId(int id) {
        this.id = id;
    }

    @NonNull
    public String getName() {
        return name;
    }

    public void setName(@NonNull String name) {
        this.name = name;
    }

    public boolean isActive() {
        return isActive;
    }

    public void setActive(boolean active) {
        isActive = active;
    }

    public void toggleActive() {
        isActive = !isActive;
    }
}
