package ch.epfl.prifiproxy.viewmodel;

import android.app.Application;
import android.arch.lifecycle.AndroidViewModel;
import android.arch.lifecycle.LiveData;
import android.support.annotation.NonNull;

import java.util.List;

import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.repository.ConfigurationRepository;

public class ConfigurationGroupViewModel extends AndroidViewModel {
    private ConfigurationRepository repository;
    private LiveData<List<ConfigurationGroup>> allGroups;

    public ConfigurationGroupViewModel(@NonNull Application application) {
        super(application);
        repository = new ConfigurationRepository(application);
        allGroups = repository.getAllGroups();
    }

    public LiveData<List<ConfigurationGroup>> getAllGroups() {
        return allGroups;
    }

    public void insert(ConfigurationGroup group) {
        repository.insert(group);
    }

    public void updateGroups(List<ConfigurationGroup> groups) {
        repository.updateGroups(groups);
    }
}
