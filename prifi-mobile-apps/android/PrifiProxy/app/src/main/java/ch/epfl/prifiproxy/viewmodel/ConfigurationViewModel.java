package ch.epfl.prifiproxy.viewmodel;

import android.app.Application;
import android.arch.lifecycle.AndroidViewModel;
import android.arch.lifecycle.LiveData;
import android.support.annotation.NonNull;

import java.util.List;

import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.repository.ConfigurationGroupRepository;
import ch.epfl.prifiproxy.repository.ConfigurationRepository;

public class ConfigurationViewModel extends AndroidViewModel {
    private ConfigurationRepository repository;
    private LiveData<List<Configuration>> configurations;
    private int groupId;

    public ConfigurationViewModel(@NonNull Application application) {
        super(application);
        repository = new ConfigurationRepository(application);
    }

    public void setGroupId(int groupId) {
        this.groupId = groupId;
        configurations = repository.getConfigurations(groupId);
    }

    public LiveData<List<Configuration>> getConfigurations() {
        return configurations;
    }

    public void insert(Configuration configuration) {
        repository.insert(configuration);
    }

    public void update(Configuration configuration) {
        repository.update(configuration);
    }

    public void delete(Configuration configuration) {
        repository.delete(configuration);
    }
}
